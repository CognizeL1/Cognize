package keeper

import (
	"encoding/json"
	"sort"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	MinRepForReward     = 20
	MinRegistrationAge = 120960
)

type RewardPool struct {
	PoolType    string           `json:"pool_type"`
	TotalAmount string           `json:"total_amount"`
	Claimed    string           `json:"claimed"`
	Epoch      uint64           `json:"epoch"`
	UpdatedAt  int64            `json:"updated_at"`
}

type RewardRecipient struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Source  string `json:"source"`
}

const (
	PoolProposer     = "proposer"
	PoolValidator   = "validator"
	PoolReputation = "reputation"
	PoolPrivacy    = "privacy"
	PoolGovernance = "governance"
	PoolService    = "service"
	PoolAIChallenge = "ai_challenge"
	PoolStaking     = "staking"
)

var RewardPools = map[string]uint64{
	PoolProposer:      2000,
	PoolValidator:    4500,
	PoolReputation:   1500,
	PoolPrivacy:      500,
	PoolGovernance:   500,
	PoolService:     500,
	PoolAIChallenge: 300,
	PoolStaking:      200,
}

func (k Keeper) InitializeRewardPools(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)

	for poolType := range RewardPools {
		key := []byte("reward/pool/" + poolType)
		if store.Get(key) == nil {
			pool := RewardPool{
				PoolType:    poolType,
				TotalAmount: "0",
				Claimed:    "0",
				Epoch:      0,
				UpdatedAt: ctx.BlockHeight(),
			}
			bz, _ := json.Marshal(&pool)
			store.Set(key, bz)
		}
	}
}

func (k Keeper) DistributeBlockReward(ctx sdk.Context, proposerVal string) error {
	blockReward := calculateBlockRewardForHeight(ctx.BlockHeight())
	if blockReward.IsZero() {
		return nil
	}

	totalBps := uint64(10000)
	proposerBps := RewardPools[PoolProposer]

	proposerReward := blockReward.Mul(math.NewInt(int64(proposerBps))).Quo(math.NewInt(int64(totalBps)))
	remainder := blockReward.Sub(proposerReward)

	k.addToRewardPool(ctx, PoolValidator, remainder)
	return nil
}

func calculateBlockRewardForHeight(height int64) math.Int {
	base, _ := math.NewIntFromString("12367000000000000000")
	if base.IsZero() {
		return math.ZeroInt()
	}

	halvings := height / 25228800
	if halvings >= 64 {
		return math.ZeroInt()
	}

	divisor := math.NewInt(1)
	for i := int64(0); i < halvings; i++ {
		divisor = divisor.Mul(math.NewInt(2))
	}

	return base.Quo(divisor)
}

func (k Keeper) addToRewardPool(ctx sdk.Context, poolType string, amount math.Int) {
	if amount.IsZero() {
		return
	}

	store := ctx.KVStore(k.storeKey)
	key := []byte("reward/pool/" + poolType)
	bz := store.Get(key)

	var pool RewardPool
	if bz != nil {
		json.Unmarshal(bz, &pool)
	}

	total, _ := math.NewIntFromString(pool.TotalAmount)
	pool.TotalAmount = total.Add(amount).String()
	pool.UpdatedAt = ctx.BlockHeight()

	bz, _ = json.Marshal(&pool)
	store.Set(key, bz)
}

func (k Keeper) DistributeEpochRewardsV2(ctx sdk.Context) error {
	if err := k.distributeReputationPoolV2(ctx); err != nil {
		return err
	}

	if err := k.distributePrivacyPoolV2(ctx); err != nil {
		return err
	}

	if err := k.distributeGovernancePoolV2(ctx); err != nil {
		return err
	}

	return nil
}

func (k Keeper) distributeReputationPoolV2(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	key := []byte("reward/pool/" + PoolReputation)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var pool RewardPool
	json.Unmarshal(bz, &pool)

	if pool.TotalAmount == "0" || pool.TotalAmount == "" {
		return nil
	}

	var eligibleAgents []types.Agent
	iter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	for iter.Valid() {
		var agent types.Agent
		k.cdc.MustUnmarshal(iter.Value(), &agent)
		if agent.Reputation >= MinRepForReward &&
			ctx.BlockHeight()-agent.RegisteredAt >= MinRegistrationAge &&
			agent.Status == types.AgentStatus_AGENT_STATUS_ONLINE {
			eligibleAgents = append(eligibleAgents, agent)
		}
		iter.Next()
	}
	iter.Close()

	if len(eligibleAgents) == 0 {
		return nil
	}

	sort.Slice(eligibleAgents, func(i, j int) bool {
		return eligibleAgents[i].Reputation > eligibleAgents[j].Reputation
	})

	totalReward, _ := math.NewIntFromString(pool.TotalAmount)
	distribution := k.calculateReputationDistributionV2(eligibleAgents, totalReward)

	for _, rec := range distribution {
		addr := sdk.MustAccAddressFromBech32(rec.Address)
		amt, _ := math.NewIntFromString(rec.Amount)
		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("cognize", amt))); err != nil {
			continue
		}
		k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("cognize", amt)))
	}

	pool.TotalAmount = "0"
	claimed, _ := math.NewIntFromString(pool.Claimed)
	pool.Claimed = claimed.Add(totalReward).String()
	bz, _ = json.Marshal(&pool)
	store.Set(key, bz)

	return nil
}

func (k Keeper) distributePrivacyPoolV2(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	key := []byte("reward/pool/" + PoolPrivacy)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var pool RewardPool
	json.Unmarshal(bz, &pool)

	if pool.TotalAmount == "0" || pool.TotalAmount == "" {
		return nil
	}

	var participants []string
	iter := store.Iterator([]byte("privacy/reward/"), []byte("privacy/reward0"))
	for iter.Valid() {
		participants = append(participants, string(iter.Key()))
		iter.Next()
	}
	iter.Close()

	if len(participants) == 0 {
		return nil
	}

	totalReward, _ := math.NewIntFromString(pool.TotalAmount)
	eachReward := totalReward.Quo(math.NewInt(int64(len(participants))))
	if eachReward.IsZero() {
		return nil
	}

	for _, p := range participants {
		addr := sdk.MustAccAddressFromBech32(p)
		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("cognize", eachReward)))
		k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("cognize", eachReward)))
	}

	pool.TotalAmount = "0"
	bz, _ = json.Marshal(&pool)
	store.Set(key, bz)

	return nil
}

func (k Keeper) distributeGovernancePoolV2(ctx sdk.Context) error {
	store := ctx.KVStore(k.storeKey)
	key := []byte("reward/pool/" + PoolGovernance)
	bz := store.Get(key)
	if bz == nil {
		return nil
	}

	var pool RewardPool
	json.Unmarshal(bz, &pool)

	if pool.TotalAmount == "0" || pool.TotalAmount == "" {
		return nil
	}

	totalReward, _ := math.NewIntFromString(pool.TotalAmount)
	if totalReward.IsZero() {
		return nil
	}

	var agents []types.Agent
	iter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	for iter.Valid() {
		var agent types.Agent
		k.cdc.MustUnmarshal(iter.Value(), &agent)
		agents = append(agents, agent)
		iter.Next()
	}
	iter.Close()

	if len(agents) == 0 {
		return nil
	}

	rewardPerAgent := totalReward.Quo(math.NewInt(int64(len(agents))))
	if rewardPerAgent.IsZero() {
		return nil
	}

	for _, agent := range agents {
		addr := sdk.MustAccAddressFromBech32(agent.Address)
		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("cognize", rewardPerAgent)))
		k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, sdk.NewCoins(sdk.NewCoin("cognize", rewardPerAgent)))
	}

	pool.TotalAmount = "0"
	bz, _ = json.Marshal(&pool)
	store.Set(key, bz)

	return nil
}

func (k Keeper) calculateReputationDistributionV2(agents []types.Agent, totalReward math.Int) []RewardRecipient {
	if len(agents) == 0 || totalReward.IsZero() {
		return nil
	}

	var distribution []RewardRecipient
	totalRep := uint64(0)
	for _, a := range agents {
		totalRep += a.Reputation
	}
	if totalRep == 0 {
		return nil
	}

	for _, agent := range agents {
		if agent.Reputation == 0 {
			continue
		}
		share := math.NewInt(int64(agent.Reputation)).Mul(totalReward).Quo(math.NewInt(int64(totalRep)))
		if share.IsZero() {
			continue
		}
		distribution = append(distribution, RewardRecipient{
			Address: agent.Address,
			Amount:  share.String(),
			Source:  PoolReputation,
		})
	}

	return distribution
}

func (k Keeper) GetRewardPoolInfoV2(ctx sdk.Context, poolType string) (*RewardPool, error) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("reward/pool/" + poolType)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrServiceNotFound
	}

	var pool RewardPool
	if err := json.Unmarshal(bz, &pool); err != nil {
		return nil, err
	}

	return &pool, nil
}

func (k Keeper) GetAllRewardPoolsV2(ctx sdk.Context) map[string]RewardPool {
	store := ctx.KVStore(k.storeKey)
	result := make(map[string]RewardPool)

	for poolType := range RewardPools {
		key := []byte("reward/pool/" + poolType)
		bz := store.Get(key)
		if bz != nil {
			var pool RewardPool
			json.Unmarshal(bz, &pool)
			result[poolType] = pool
		}
	}

	return result
}