package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	StablecoinFeeBps      = 25
	MinCognizeDeposit     = 100
	OraclePriceDeviation  = 500
	ParamsUpdateInterval = 20160
)

func (k Keeper) ProcessStablecoinDeposit(ctx sdk.Context, depositor string, cognizeAmount string, denom string) (string, error) {
	amount, ok := math.NewIntFromString(cognizeAmount)
	if !ok || amount.LT(math.NewInt(MinCognizeDeposit)) {
		return "", types.ErrDepositTooLow
	}

	stableDenom := denom
	if stableDenom == "" {
		stableDenom = types.StablecoinCUSD
	}

	deposit := types.StablecoinDeposit{
		Depositor:    depositor,
		Amount:    cognizeAmount,
		LockedCognize: cognizeAmount,
		DepositTime: ctx.BlockTime().Unix(),
		Withdrawn: false,
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&deposit)
	store.Set(types.KeyStablecoinDeposit(stableDenom, depositor), bz)

	stableAmount := amount.Mul(math.NewInt(10000)).Quo(k.getOraclePrice(ctx, stableDenom))

	typesMsg := types.StablecoinSwap{
		SwapID:        fmt.Sprintf("swap-%d-%s", ctx.BlockHeight(), depositor[:8]),
		Depositor:     depositor,
		CognizeAmount: cognizeAmount,
		StableAmount: stableAmount.String(),
		Rate:         k.getOraclePrice(ctx, stableDenom).String(),
		Direction:    "deposit",
		Status:       "completed",
		BlockHeight:  ctx.BlockHeight(),
		BlockTime:   ctx.BlockTime().Unix(),
	}

	bz, _ = json.Marshal(&typesMsg)
	store.Set(types.KeyStablecoinSwap(typesMsg.SwapID), bz)

	return stableAmount.String(), nil
}

func (k Keeper) ProcessStablecoinWithdrawal(ctx sdk.Context, depositor, stableAmount, denom string) (string, error) {
	stableDenom := denom
	if stableDenom == "" {
		stableDenom = types.StablecoinCUSD
	}

	store := ctx.KVStore(k.storeKey)
	depositKey := types.KeyStablecoinDeposit(stableDenom, depositor)
	bz := store.Get(depositKey)
	if bz == nil {
		return "", types.ErrStablecoinNotFound
	}

	var deposit types.StablecoinDeposit
	json.Unmarshal(bz, &deposit)

	if deposit.Withdrawn {
		return "", types.ErrStablecoinPaused
	}

	amount, _ := math.NewIntFromString(stableAmount)
	depositInt, _ := math.NewIntFromString(deposit.Amount)

	if amount.GT(depositInt) {
		return "", types.ErrWithdrawalExceedsBalance
	}

	oraclePrice := k.getOraclePrice(ctx, stableDenom)
	cognizeAmount := amount.Mul(oraclePrice).Quo(math.NewInt(10000))
	fee := cognizeAmount.Mul(math.NewInt(StablecoinFeeBps)).Quo(math.NewInt(10000))
	payout := cognizeAmount.Sub(fee)

	deposit.Amount = depositInt.Sub(amount).String()
	bz, _ = json.Marshal(&deposit)
	store.Set(depositKey, bz)

	swap := types.StablecoinSwap{
		SwapID:        fmt.Sprintf("swap-%d-%s", ctx.BlockHeight(), depositor[:8]),
		Depositor:     depositor,
		CognizeAmount: payout.String(),
		StableAmount: stableAmount,
		Rate:         oraclePrice.String(),
		Direction:    "withdrawal",
		Status:      "completed",
		BlockHeight: ctx.BlockHeight(),
		BlockTime:  ctx.BlockTime().Unix(),
	}

	bz, _ = json.Marshal(&swap)
	store.Set(types.KeyStablecoinSwap(swap.SwapID), bz)

	return payout.String(), nil
}

func (k Keeper) getOraclePrice(ctx sdk.Context, denom string) math.Int {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyStablecoin(denom)
	bz := store.Get(key)

	if bz == nil {
		return math.NewInt(10000)
	}

	var pool types.Stablecoin
	json.Unmarshal(bz, &pool)

	price, _ := math.NewIntFromString(pool.OraclePrice)
	if price.IsZero() {
		return math.NewInt(10000)
	}

	return price
}

func (k Keeper) GetStablecoinPool(ctx sdk.Context, denom string) (*types.Stablecoin, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyStablecoin(denom)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrStablecoinNotFound
	}

	var pool types.Stablecoin
	if err := json.Unmarshal(bz, &pool); err != nil {
		return nil, err
	}

	return &pool, nil
}

func (k Keeper) SetStablecoinPool(ctx sdk.Context, pool types.Stablecoin) error {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&pool)
	store.Set(types.KeyStablecoin(pool.Denom), bz)
	return nil
}

func (k Keeper) CalculateDynamicParams(ctx sdk.Context) types.DynamicParams {
	store := ctx.KVStore(k.storeKey)

	agentIter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	var activeAgents uint64
	for agentIter.Valid() {
		activeAgents++
		agentIter.Next()
	}
	agentIter.Close()

	totalStake := k.GetTotalStaked(ctx)
	transactions24h := k.GetTransactions24h(ctx)
	gasUsage24h := k.GetGasUsage24h(ctx)

	networkImportance := calculateNetworkImportance(totalStake, activeAgents, transactions24h, gasUsage24h)

	baseStake := uint64(10)
	minStake := baseStake
	if networkImportance > 1000 {
		minStake = baseStake + (networkImportance-1000)/100
		if minStake > 100 {
			minStake = 100
		}
	}

	burnAdjustment := uint64(2)
	if networkImportance > 500 {
		burnAdjustment = 2 + (networkImportance-500)/200
		if burnAdjustment > 10 {
			burnAdjustment = 10
		}
	}

	return types.DynamicParams{
		NetworkImportance: networkImportance,
		TotalStake:        totalStake.String(),
		ActiveAgents:      activeAgents,
		Transactions24h:  transactions24h,
		GasUsage24h:       gasUsage24h,
		MinStake:         baseStake,
		MinStakeDynamic:  minStake,
		RegisterBurn:    burnAdjustment,
		DeployBurn:      burnAdjustment / 2,
		EpochLength:     720,
		LastUpdate:     ctx.BlockTime().Unix(),
	}
}

func calculateNetworkImportance(totalStake math.Int, activeAgents, tx24h, gas24h uint64) uint64 {
	imp := totalStake.BigInt().Uint64() / 1e18
	imp += activeAgents * 100
	imp += tx24h / 100
	imp += gas24h / 1e6
	return imp
}

func (k Keeper) GetTotalStaked(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))

	var total math.Int
	for iter.Valid() {
		var agent types.Agent
		k.cdc.MustUnmarshal(iter.Value(), &agent)
		total = total.Add(agent.StakeAmount.Amount)
		iter.Next()
	}
	iter.Close()

	return total
}

func (k Keeper) GetTransactions24h(ctx sdk.Context) uint64 {
	key := []byte("txcount_24h")
	bz := ctx.KVStore(k.storeKey).Get(key)
	if bz == nil {
		return 0
	}
	return types.BytesToUint64(bz)
}

func (k Keeper) GetGasUsage24h(ctx sdk.Context) uint64 {
	key := []byte("gas_24h")
	bz := ctx.KVStore(k.storeKey).Get(key)
	if bz == nil {
		return 0
	}
	return types.BytesToUint64(bz)
}

func (k Keeper) RecordTransaction(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("txcount_24h")
	bz := store.Get(key)

	var count uint64
	if bz != nil {
		count = types.BytesToUint64(bz)
	}
	count++

	store.Set(key, types.Uint64ToBytes(count))
}

func (k Keeper) RecordGasUsage(ctx sdk.Context, gasUsed uint64) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("gas_24h")
	bz := store.Get(key)

	var total uint64
	if bz != nil {
		total = types.BytesToUint64(bz)
	}
	total += gasUsed

	store.Set(key, types.Uint64ToBytes(total))
}

func (k Keeper) GetInsurancePool(ctx sdk.Context, agent string) (*types.InsurancePool, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyInsurancePool(agent)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrAgentNotFound
	}

	var pool types.InsurancePool
	if err := json.Unmarshal(bz, &pool); err != nil {
		return nil, err
	}

	return &pool, nil
}

func (k Keeper) CreateInsurancePool(ctx sdk.Context, agent, coverAmount, premium string, durationBlocks int64) error {
	pool := types.InsurancePool{
		PoolID:      fmt.Sprintf("pool-%s-%d", agent[:8], ctx.BlockHeight()),
		Agent:      agent,
		CoverAmount: coverAmount,
		Premium:   premium,
		StartBlock: ctx.BlockHeight(),
		EndBlock:  ctx.BlockHeight() + durationBlocks,
		Active:    true,
		ClaimsPaid: "0",
		Status:    "active",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&pool)
	store.Set(types.KeyInsurancePool(agent), bz)

	return nil
}

func (k Keeper) SubmitInsuranceClaim(ctx sdk.Context, poolID, agent, amount, reason string) (string, error) {
	claimID := fmt.Sprintf("claim-%d-%s", ctx.BlockHeight(), agent[:8])

	claim := types.InsuranceClaim{
		ClaimID:    claimID,
		PoolID:    poolID,
		Agent:     agent,
		Amount:    amount,
		Reason:   reason,
		BlockTime: ctx.BlockTime().Unix(),
		Status:   "pending",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&claim)
	store.Set(types.KeyInsuranceClaim(claimID), bz)

	return claimID, nil
}

func (k Keeper) ProcessInsuranceClaim(ctx sdk.Context, claimID, approver string, approved bool) error {
	store := ctx.KVStore(k.storeKey)
	key := types.KeyInsuranceClaim(claimID)
	bz := store.Get(key)
	if bz == nil {
		return types.ErrAgentNotFound
	}

	var claim types.InsuranceClaim
	json.Unmarshal(bz, &claim)

	if claim.Status != "pending" {
		return types.ErrStablecoinPaused
	}

	if approved {
		claim.Status = "approved"
		claim.ApprovedBy = approver
		claim.PaidAt = ctx.BlockHeight()

		claimAmt, _ := math.NewIntFromString(claim.Amount)
		fee := claimAmt.Mul(math.NewInt(StablecoinFeeBps)).Quo(math.NewInt(10000))
		payout := claimAmt.Sub(fee)

		agentAddr := sdk.MustAccAddressFromBech32(claim.Agent)
		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("acognize", payout))); err != nil {
			return err
		}
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, agentAddr, sdk.NewCoins(sdk.NewCoin("acognize", payout))); err != nil {
			return err
		}
	} else {
		claim.Status = "rejected"
	}

	bz, _ = json.Marshal(&claim)
	store.Set(key, bz)

	return nil
}

func (k Keeper) GetAgentRanking(ctx sdk.Context, limit int) []types.Agent {
	store := ctx.KVStore(k.storeKey)

	var allAgents []types.Agent
	iter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	for iter.Valid() {
		var agent types.Agent
		k.cdc.MustUnmarshal(iter.Value(), &agent)
		allAgents = append(allAgents, agent)
		iter.Next()
	}
	iter.Close()

	for i := 0; i < len(allAgents)-1; i++ {
		for j := i + 1; j < len(allAgents); j++ {
			if allAgents[j].Reputation > allAgents[i].Reputation {
				allAgents[i], allAgents[j] = allAgents[j], allAgents[i]
			}
		}
	}

	if limit > 0 && len(allAgents) > limit {
		allAgents = allAgents[:limit]
	}

	return allAgents
}