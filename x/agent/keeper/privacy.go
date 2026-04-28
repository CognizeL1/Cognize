package keeper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	PrivacyKeyFeeBps        = 10
	PrivacyMixFeeBps        = 30
	PrivacyMixMinDeposit    = 100
	PrivacyMixMaxParticipants = 100
	PrivacyMixEpochBlocks  = 720
	PrivacyRewardBps       = 5
)

func (k Keeper) GeneratePrivacyAccessKey(ctx sdk.Context, issuer, resourceType, resourceID, accessLevel string, maxUses uint64, durationBlocks int64) (string, error) {
	if resourceType == "" || resourceID == "" {
		return "", types.ErrInvalidKey
	}

	keyBytes := make([]byte, 32)
	rand.Read(keyBytes)
	key := hex.EncodeToString(keyBytes)

	expiresAt := ctx.BlockHeight() + durationBlocks
	nonce := ctx.BlockHeight()

	keyID := fmt.Sprintf("pkey-%s-%s-%s-%d", issuer[:8], resourceType, resourceID[:8], nonce)

	privacyKey := types.PrivacyKey{
		KeyID:        keyID,
		Issuer:       issuer,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Key:          key,
		AccessLevel:  accessLevel,
		MaxUses:      maxUses,
		UsedCount:   0,
		ExpiresAt:   expiresAt,
		Revoked:     false,
		CreatedAt:   ctx.BlockHeight(),
		Metadata:    "",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&privacyKey)
	store.Set(types.KeyPrivacyKey(keyID), bz)
	store.Set(types.KeyPrivacyKeyByResource(resourceType, resourceID), bz)

	k.recordRevenue(ctx, issuer, math.NewInt(PrivacyKeyFeeBps))

	return keyID + ":" + key, nil
}

func (k Keeper) ValidatePrivacyKey(ctx sdk.Context, keyIDWithKey, user string) error {
	parts := splitKeyID(keyIDWithKey)
	if len(parts) != 2 {
		return types.ErrInvalidKey
	}

	keyID, keyValue := parts[0], parts[1]

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrivacyKey(keyID))
	if bz == nil {
		return types.ErrPrivacyKeyNotFound
	}

	var privacyKey types.PrivacyKey
	json.Unmarshal(bz, &privacyKey)

	if privacyKey.Revoked {
		return types.ErrPrivacyKeyRevoked
	}

	if ctx.BlockHeight() > privacyKey.ExpiresAt {
		return types.ErrPrivacyKeyExpired
	}

	if privacyKey.Key != keyValue {
		return types.ErrInvalidKey
	}

	if privacyKey.UsedCount >= privacyKey.MaxUses && privacyKey.MaxUses > 0 {
		return types.ErrKeyAlreadyUsed
	}

	privacyKey.UsedCount++
	bz, _ = json.Marshal(&privacyKey)
	store.Set(types.KeyPrivacyKey(keyID), bz)

	usage := types.PrivacyKeyUsage{
		KeyID:        keyID,
		User:        user,
		UsedAt:      ctx.BlockTime().Unix(),
		BlockHeight: ctx.BlockHeight(),
	}
	bz, _ = json.Marshal(&usage)
	store.Set([]byte("privacy/usage/"+keyID+"/"+user), bz)

	return nil
}

func (k Keeper) RevokePrivacyKey(ctx sdk.Context, keyID, revoker string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrivacyKey(keyID))
	if bz == nil {
		return types.ErrPrivacyKeyNotFound
	}

	var privacyKey types.PrivacyKey
	json.Unmarshal(bz, &privacyKey)

	if privacyKey.Issuer != revoker {
		return types.ErrUnauthorized
	}

	privacyKey.Revoked = true
	bz, _ = json.Marshal(&privacyKey)
	store.Set(types.KeyPrivacyKey(keyID), bz)

	return nil
}

func (k Keeper) RegisterPrivateService(ctx sdk.Context, agent, name, accessType, price, accessKeyPrice string, maxAccess uint64) (string, error) {
	if accessType == types.AccessLevelPrivate || accessType == types.AccessLevelTokenGated || accessType == types.AccessLevelWhitelist {
		serviceID := fmt.Sprintf("private-%d-%s", ctx.BlockHeight(), agent[:8])

		privateService := types.PrivateService{
			ServiceID:       serviceID,
			AgentAddress:    agent,
			Name:           name,
			IsPrivate:      true,
			AccessType:     accessType,
			Price:        price,
			AccessKeyPrice: accessKeyPrice,
			MaxAccess:    maxAccess,
			CurrentAccess: 0,
			Status:      "active",
		}

		store := ctx.KVStore(k.storeKey)
		bz, _ := json.Marshal(&privateService)
		store.Set(types.KeyPrivateService(serviceID), bz)

		return serviceID, nil
	}

	return "", types.ErrServicePrivate
}

func (k Keeper) PurchaseServiceAccessKey(ctx sdk.Context, serviceID, buyer, paymentAmount string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrivateService(serviceID))
	if bz == nil {
		return "", types.ErrServiceNotFound
	}

	var service types.PrivateService
	json.Unmarshal(bz, &service)

	accessKeyPrice, _ := math.NewIntFromString(service.AccessKeyPrice)
	payAmt, _ := math.NewIntFromString(paymentAmount)

	if payAmt.LT(accessKeyPrice) {
		return "", types.ErrInsufficientStake
	}

	if service.CurrentAccess >= service.MaxAccess && service.MaxAccess > 0 {
		return "", types.ErrServicePrivate
	}

	keyID, err := k.GeneratePrivacyAccessKey(ctx, service.AgentAddress, "service", serviceID, service.AccessType, 1, 20160)
	if err != nil {
		return "", err
	}

	service.CurrentAccess++
	bz, _ = json.Marshal(&service)
	store.Set(types.KeyPrivateService(serviceID), bz)

	payAmt, _ = math.NewIntFromString(paymentAmount)
	fee := payAmt.Mul(math.NewInt(PrivacyKeyFeeBps)).Quo(math.NewInt(10000))
	sellerAmt := payAmt.Sub(fee)

	sellerAddr := sdk.MustAccAddressFromBech32(service.AgentAddress)
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("acognize", sellerAmt))); err != nil {
		return "", err
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sellerAddr, sdk.NewCoins(sdk.NewCoin("acognize", sellerAmt))); err != nil {
		return "", err
	}

	return keyID, nil
}

func (k Keeper) GetPrivateService(ctx sdk.Context, serviceID string) (*types.PrivateService, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrivateService(serviceID))
	if bz == nil {
		return nil, types.ErrServiceNotFound
	}

	var service types.PrivateService
	if err := json.Unmarshal(bz, &service); err != nil {
		return nil, err
	}

	return &service, nil
}

func (k Keeper) CreatePrivacyMix(ctx sdk.Context, creator, denomination, minDeposit string, maxParticipants uint64) (string, error) {
	deposit, _ := math.NewIntFromString(minDeposit)
	if deposit.LT(math.NewInt(PrivacyMixMinDeposit)) {
		return "", types.ErrDepositTooLow
	}

	poolID := fmt.Sprintf("mix-%d-%s", ctx.BlockHeight(), creator[:8])

	privacyPool := types.PrivacyPool{
		PoolID:         poolID,
		Denomination:   denomination,
		TotalDeposited: "0",
		TotalWithdrawn: "0",
		Status:        "open",
		Epoch:        0,
		Participants: 0,
		FeeBps:       PrivacyMixFeeBps,
		MinDeposit:    minDeposit,
		StartBlock:   ctx.BlockHeight(),
		EndBlock:     ctx.BlockHeight() + PrivacyMixEpochBlocks,
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&privacyPool)
	store.Set(types.KeyPrivacyPool(poolID), bz)

	return poolID, nil
}

func (k Keeper) CommitToMix(ctx sdk.Context, poolID, depositor, amount string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrivacyPool(poolID))
	if bz == nil {
		return "", types.ErrMixNotFound
	}

	var pool types.PrivacyPool
	json.Unmarshal(bz, &pool)

	if pool.Status != "open" {
		return "", types.ErrMixInProgress
	}

	if ctx.BlockHeight() > pool.EndBlock {
		return "", types.ErrMixPhaseWrong
	}

	if pool.Participants >= PrivacyMixMaxParticipants {
		return "", types.ErrMixInProgress
	}

	commitment := generateCommitment(depositor, poolID, amount, ctx.BlockHeight())

	commit := types.MixCommitment{
		Commitment: commitment,
		Hash:        sha256Hash(commitment),
		Deposit:    amount,
		Depositor:  depositor,
		Status:     "committed",
		Block:     ctx.BlockHeight(),
		LeafIndex: pool.Participants,
	}

	bz, _ = json.Marshal(&commit)
	store.Set(types.KeyMixCommitment(poolID, commitment), bz)

	pool.Participants++
	deposited, _ := math.NewIntFromString(pool.TotalDeposited)
	amountInt, _ := math.NewIntFromString(amount)
	pool.TotalDeposited = deposited.Add(amountInt).String()
	bz, _ = json.Marshal(&pool)
	store.Set(types.KeyPrivacyPool(poolID), bz)

	return commitment, nil
}

func (k Keeper) WithdrawFromMix(ctx sdk.Context, poolID, recipient, commitment, proof string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyMixCommitment(poolID, commitment))
	if bz == nil {
		return types.ErrCommitmentNotFound
	}

	var commit types.MixCommitment
	json.Unmarshal(bz, &commit)

	if commit.Status != "committed" {
		return types.ErrDoubleSpend
	}

	commit.Status = "withdrawn"
	bz, _ = json.Marshal(&commit)
	store.Set(types.KeyMixCommitment(poolID, commitment), bz)

	recipientAddr := sdk.MustAccAddressFromBech32(recipient)
	rewardAmt, _ := math.NewIntFromString(commit.Deposit)
	fee := rewardAmt.Mul(math.NewInt(PrivacyMixFeeBps)).Quo(math.NewInt(10000))
	payout := rewardAmt.Sub(fee)

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("acognize", payout))); err != nil {
		return err
	}
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipientAddr, sdk.NewCoins(sdk.NewCoin("acognize", payout))); err != nil {
		return err
	}

	poolBz := store.Get(types.KeyPrivacyPool(poolID))
	if poolBz != nil {
		var pool types.PrivacyPool
		json.Unmarshal(poolBz, &pool)
		withdrawn, _ := math.NewIntFromString(pool.TotalWithdrawn)
		pool.TotalWithdrawn = withdrawn.Add(payout).String()
		poolBz, _ = json.Marshal(&pool)
		store.Set(types.KeyPrivacyPool(poolID), poolBz)
	}

	k.recordPrivacyReward(ctx, recipient)

	return nil
}

func (k Keeper) recordPrivacyReward(ctx sdk.Context, user string) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("privacy/reward/" + user)

	var reward struct {
		Total   string `json:"total"`
		Count   uint64 `json:"count"`
		Period  int64  `json:"period"`
	}
	bz := store.Get(key)
	if bz != nil {
		json.Unmarshal(bz, &reward)
	}

	currentPeriod := ctx.BlockHeight() / 20160
	if reward.Period != currentPeriod {
		reward.Total = "0"
		reward.Count = 0
		reward.Period = currentPeriod
	}

	rewardAmt := math.NewInt(PrivacyRewardBps)
	rewardTotal, _ := math.NewIntFromString(reward.Total)
	reward.Total = rewardTotal.Add(rewardAmt).String()
	reward.Count++

	bz, _ = json.Marshal(&reward)
	store.Set(key, bz)
}

func (k Keeper) GetPrivacyStats(ctx sdk.Context) map[string]interface{} {
	store := ctx.KVStore(k.storeKey)

	var totalKeys, activeKeys, totalMixes, activeMixes uint64

	keyIter := store.Iterator([]byte("privacy/key/"), []byte("privacy/key0"))
	for keyIter.Valid() {
		totalKeys++
		keyIter.Next()
	}
	keyIter.Close()

	mixIter := store.Iterator([]byte("privacy/pool/"), []byte("privacy/pool0"))
	for mixIter.Valid() {
		totalMixes++
		mixIter.Next()
	}
	mixIter.Close()

	return map[string]interface{}{
		"total_privacy_keys":  totalKeys,
		"active_keys":        activeKeys,
		"total_mixes":        totalMixes,
		"active_mixes":      activeMixes,
		"block_height":     ctx.BlockHeight(),
	}
}

func (k Keeper) AntiManipulationCheck(ctx sdk.Context, actor string, action string) error {
	store := ctx.KVStore(k.storeKey)

	key := []byte("rate_limit/" + actor + "/" + action)
	bz := store.Get(key)

	var count uint64
	if bz != nil {
		count = types.BytesToUint64(bz)
	}

	maxActions := map[string]uint64{
		"register":  10,
		"heartbeat": 100,
		"transfer": 50,
		"mix_commit": 5,
	}

	maxAllowed := maxActions[action]
	if maxAllowed == 0 {
		maxAllowed = 20
	}

	if count >= maxAllowed {
		return types.ErrTooManyRequests
	}

	count++
	store.Set(key, types.Uint64ToBytes(count))

	currentWindow := uint64(ctx.BlockHeight() / 100)
	windowKey := []byte("rate_limit/" + actor + "/" + action + "/window")
	store.Set(windowKey, types.Uint64ToBytes(currentWindow))

	return nil
}

func generateCommitment(depositor, poolID, amount string, block int64) string {
	data := fmt.Sprintf("%s-%s-%s-%d", depositor, poolID, amount, block)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func sha256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func splitKeyID(s string) []string {
	var result []string
	var current string
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(s[i])
		}
	}
	result = append(result, current)
	return result
}