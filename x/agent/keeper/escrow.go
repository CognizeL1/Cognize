package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	EscrowFeeBps = 25
)

type EscrowStore struct {
	EscrowID     string `json:"escrow_id"`
	Seller      string `json:"seller"`
	Buyer       string `json:"buyer"`
	Amount      string `json:"amount"`
	ServiceID   string `json:"service_id"`
	TaskID      string `json:"task_id"`
	Status     string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt  int64  `json:"expires_at"`
	CompletedAt int64 `json:"completed_at"`
	Memo       string `json:"memo"`
}

func (k Keeper) CreateEscrow(ctx sdk.Context, seller, buyer, amount, serviceID, taskID, memo string) (string, error) {
	amountInt, ok := math.NewIntFromString(amount)
	if !ok || amountInt.IsZero() || amountInt.IsNegative() {
		return "", types.ErrInvalidAmount
	}

	buyerAgent, found := k.GetAgent(ctx, buyer)
	if !found {
		return "", types.ErrAgentNotFound
	}

	if buyerAgent.StakeAmount.Amount.LT(amountInt) {
		return "", types.ErrInsufficientFunds
	}

	escrowID := fmt.Sprintf("escrow-%d-%s-%s", ctx.BlockHeight(), buyer, seller[:8])

	escrow := EscrowStore{
		EscrowID:   escrowID,
		Seller:    seller,
		Buyer:    buyer,
		Amount:   amount,
		Status:   "funded",
		ServiceID: serviceID,
		TaskID:  taskID,
		Memo:    memo,
		CreatedAt: ctx.BlockHeight(),
		ExpiresAt: ctx.BlockHeight() + 20160,
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&escrow)
	store.Set(types.KeyEscrow(escrowID), bz)

	return escrowID, nil
}

func (k Keeper) ConfirmDelivery(ctx sdk.Context, escrowID, deliverer string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEscrow(escrowID))
	if bz == nil {
		return types.ErrEscrowNotFound
	}

	var escrow EscrowStore
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return err
	}

	if escrow.Seller != deliverer {
		return types.ErrUnauthorizedParty
	}

	escrow.Status = "delivered"
	bz, _ = json.Marshal(&escrow)
	store.Set(types.KeyEscrow(escrowID), bz)

	return nil
}

func (k Keeper) CompleteEscrow(ctx sdk.Context, escrowID, completer string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEscrow(escrowID))
	if bz == nil {
		return types.ErrEscrowNotFound
	}

	var escrow EscrowStore
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return err
	}

	if escrow.Status == "completed" {
		return types.ErrEscrowAlreadyCompleted
	}
	if escrow.Status == "refunded" {
		return types.ErrEscrowRefunded
	}

	if ctx.BlockHeight() > escrow.ExpiresAt {
		escrow.Status = "expired"
		bz, _ = json.Marshal(&escrow)
		store.Set(types.KeyEscrow(escrowID), bz)
		return types.ErrEscrowExpired
	}

	amountInt, _ := math.NewIntFromString(escrow.Amount)
	fee := amountInt.Mul(math.NewInt(EscrowFeeBps)).Quo(math.NewInt(10000))

	escrow.Status = "completed"
	escrow.CompletedAt = ctx.BlockHeight()
	bz, _ = json.Marshal(&escrow)
	store.Set(types.KeyEscrow(escrowID), bz)

	k.recordRevenue(ctx, escrow.Seller, amountInt.Sub(fee))
	k.recordRevenue(ctx, "fee_collector", fee)

	return nil
}

func (k Keeper) OpenDispute(ctx sdk.Context, escrowID, opener, reason string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEscrow(escrowID))
	if bz == nil {
		return types.ErrEscrowNotFound
	}

	var escrow EscrowStore
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return err
	}

	if escrow.Status != "delivered" && escrow.Status != "funded" {
		return types.ErrDisputeAlreadyOpen
	}

	escrow.Status = "disputed"
	bz, _ = json.Marshal(&escrow)
	store.Set(types.KeyEscrow(escrowID), bz)

	return nil
}

func (k Keeper) RefundExpiredEscrow(ctx sdk.Context, escrowID string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEscrow(escrowID))
	if bz == nil {
		return types.ErrEscrowNotFound
	}

	var escrow EscrowStore
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return err
	}

	if escrow.Status != "funded" && escrow.Status != "delivered" {
		return types.ErrEscrowRefunded
	}

	escrow.Status = "refunded"
	bz, _ = json.Marshal(&escrow)
	store.Set(types.KeyEscrow(escrowID), bz)

	return nil
}

func (k Keeper) GetEscrow(ctx sdk.Context, escrowID string) (*EscrowStore, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyEscrow(escrowID))
	if bz == nil {
		return nil, types.ErrEscrowNotFound
	}

	var escrow EscrowStore
	if err := json.Unmarshal(bz, &escrow); err != nil {
		return nil, err
	}

	return &escrow, nil
}

func (k Keeper) recordRevenue(ctx sdk.Context, agentAddr string, amount math.Int) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("revenue/" + agentAddr)

	var revenue struct {
		Total   string `json:"total"`
		Period  int64  `json:"period"`
	}
	bz := store.Get(key)
	if bz != nil {
		json.Unmarshal(bz, &revenue)
	}

	currentPeriod := ctx.BlockHeight() / 20160
	if revenue.Period != currentPeriod {
		revenue.Total = "0"
		revenue.Period = currentPeriod
	}

	revenueInt, _ := math.NewIntFromString(revenue.Total)
	revenue.Total = revenueInt.Add(amount).String()

	bz, _ = json.Marshal(&revenue)
	store.Set(key, bz)
}

func (k Keeper) GetAgentRevenue(ctx sdk.Context, agentAddr string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("revenue/" + agentAddr)
	bz := store.Get(key)
	if bz == nil {
		return "0", nil
	}

	var revenue struct {
		Total string `json:"total"`
	}
	json.Unmarshal(bz, &revenue)
	return revenue.Total, nil
}

func (k Keeper) RecordServiceMetrics(ctx sdk.Context, serviceID string, success bool, latencyMs uint64) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("metrics/" + serviceID)

	var metrics struct {
		TotalCalls      uint64 `json:"total_calls"`
		SuccessfulCalls uint64 `json:"successful_calls"`
		FailedCalls     uint64 `json:"failed_calls"`
		UptimePercent   uint64 `json:"uptime_percent"`
		AvgLatencyMs    uint64 `json:"avg_latency_ms"`
		LastCheckAt    int64  `json:"last_check_at"`
	}
	bz := store.Get(key)
	if bz != nil {
		json.Unmarshal(bz, &metrics)
	}

	metrics.TotalCalls++
	if success {
		metrics.SuccessfulCalls++
	} else {
		metrics.FailedCalls++
	}

	if metrics.TotalCalls > 0 {
		metrics.UptimePercent = (metrics.SuccessfulCalls * 10000) / metrics.TotalCalls
	}

	avgLatency := (metrics.AvgLatencyMs*(metrics.TotalCalls-1) + latencyMs) / metrics.TotalCalls
	metrics.AvgLatencyMs = avgLatency
	metrics.LastCheckAt = ctx.BlockHeight()

	bz, _ = json.Marshal(&metrics)
	store.Set(key, bz)
}

func (k Keeper) GetServiceMetrics(ctx sdk.Context, serviceID string) (map[string]interface{}, error) {
	store := ctx.KVStore(k.storeKey)
	key := []byte("metrics/" + serviceID)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrServiceNotFound
	}

	var metrics map[string]interface{}
	if err := json.Unmarshal(bz, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

func (k Keeper) GetNetworkMetrics(ctx sdk.Context) types.NetworkMetrics {
	store := ctx.KVStore(k.storeKey)

	agentIter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	var totalAgents, activeAgents uint64
	for agentIter.Valid() {
		totalAgents++
		agentIter.Next()
	}
	agentIter.Close()

	escrowIter := store.Iterator([]byte("escrow/"), []byte("escrow0"))
	var activeEscrows uint64
	for escrowIter.Valid() {
		activeEscrows++
		escrowIter.Next()
	}
	escrowIter.Close()

	return types.NetworkMetrics{
		TotalAgents:   totalAgents,
		ActiveAgents:  activeAgents,
		TotalEscrows:  activeEscrows,
		BlockHeight:   ctx.BlockHeight(),
		Timestamp:    ctx.BlockTime().Unix(),
	}
}

func (k Keeper) GetTopAgents(ctx sdk.Context, limit int) []types.Agent {
	store := ctx.KVStore(k.storeKey)

	var agents []types.Agent
	iter := store.Iterator([]byte(types.AgentKeyPrefix), []byte(types.AgentKeyPrefix+"/z"))
	for iter.Valid() && len(agents) < limit {
		var agent types.Agent
		k.cdc.MustUnmarshal(iter.Value(), &agent)
		agents = append(agents, agent)
		iter.Next()
	}
	iter.Close()

	for i := 0; i < len(agents)-1; i++ {
		for j := i + 1; j < len(agents); j++ {
			if agents[j].Reputation > agents[i].Reputation {
				agents[i], agents[j] = agents[j], agents[i]
			}
		}
	}

	return agents
}