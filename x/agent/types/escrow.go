package types

import (
	"fmt"

	"cosmossdk.io/errors"
)

var (
	ErrEscrowNotFound         = errors.Register(ModuleName, 1200, "escrow not found")
	ErrEscrowAlreadyCompleted   = errors.Register(ModuleName, 1201, "escrow already completed")
	ErrEscrowRefunded        = errors.Register(ModuleName, 1202, "escrow already refunded")
	ErrEscrowDisputed       = errors.Register(ModuleName, 1203, "escrow is disputed")
	ErrEscrowExpired        = errors.Register(ModuleName, 1204, "escrow expired")
	ErrUnauthorizedParty   = errors.Register(ModuleName, 1205, "unauthorized party")
	ErrInsufficientFunds   = errors.Register(ModuleName, 1206, "insufficient funds for escrow")
	ErrInvalidAmount        = errors.Register(ModuleName, 1207, "invalid escrow amount")
	ErrDisputeAlreadyOpen   = errors.Register(ModuleName, 1208, "dispute already open")
	ErrDisputeNotOpen      = errors.Register(ModuleName, 1209, "no dispute open")
	ErrArbitrationNotResolved = errors.Register(ModuleName, 1210, "arbitration not resolved")
	ErrStablecoinConversion = errors.Register(ModuleName, 1211, "stablecoin conversion failed")
)

type EscrowStatus int32

const (
	EscrowStatus_PENDING EscrowStatus = iota
	EscrowStatus_FUNDED
	EscrowStatus_DELIVERED
	EscrowStatus_COMPLETED
	EscrowStatus_DISPUTED
	EscrowStatus_REFUNDED
	EscrowStatus_EXPIRED
)

type Escrow struct {
	EscrowID        string        `json:"escrow_id"`
	Seller         string        `json:"seller"`
	Buyer          string        `json:"buyer"`
	Amount         string        `json:"amount"`
	StableAmount   string        `json:"stable_amount"`
	ServiceID     string        `json:"service_id"`
	TaskID        string        `json:"task_id"`
	Status        EscrowStatus   `json:"status"`
	CreatedAt     int64         `json:"created_at"`
	ExpiresAt     int64         `json:"expires_at"`
	CompletedAt   int64         `json:"completed_at"`
	DisputeReason string        `json:"dispute_reason"`
	DisputeOpenBy string      `json:"dispute_open_by"`
	DisputeOpenAt int64        `json:"dispute_open_at"`
	ArbitrationDecision string   `json:"arbitration_decision"`
	ArbitrationWinner string   `json:"arbitration_winner"`
	FeePaid       string        `json:"fee_paid"`
	Memo          string        `json:"memo"`
}

func (e Escrow) String() string {
	return fmt.Sprintf("Escrow %s: %s -> %s (%s) [%s]", 
		e.EscrowID, e.Buyer, e.Seller, e.Amount, e.Status.String())
}

func (e EscrowStatus) String() string {
	return [...]string{
		"PENDING",
		"FUNDED",
		"DELIVERED",
		"COMPLETED",
		"DISPUTED",
		"REFUNDED",
		"EXPIRED",
	}[e]
}

func KeyEscrow(id string) []byte {
	return []byte("escrow/" + id)
}

func KeyEscrowBySeller(seller string) []byte {
	return []byte("escrow/seller/" + seller)
}

func KeyEscrowByBuyer(buyer string) []byte {
	return []byte("escrow/buyer/" + buyer)
}

type StablecoinPool struct {
	TotalDeposited    string `json:"total_deposited"`
	TotalWithdrawn  string `json:"total_withdrawn"`
	CurrentSupply string `json:"current_supply"`
	OraclePrice   string `json:"oracle_price"`
	LastUpdate   int64  `json:"last_update"`
}

type ServiceSLA struct {
	ServiceID       string `json:"service_id"`
	AgentAddress   string `json:"agent_address"`
	UptimePercent  uint64 `json:"uptime_percent"`
	TotalCalls    uint64 `json:"total_calls"`
	SuccessfulCalls uint64 `json:"successful_calls"`
	AvgLatencyMs  uint64 `json:"avg_latency_ms"`
	LastCheckAt   int64  `json:"last_check_at"`
	PenaltyAccumulated string `json:"penalty_accumulated"`
	StreakCount  int64  `json:"streak_count"`
	SlaTier     string `json:"sla_tier"`
}

const (
	SlaUptimeBasic      uint64 = 9500
	SlaUptimeStandard  uint64 = 9900
	SlaUptimePremium  uint64 = 9990
	SlaUptimeEnterprise uint64 = 9999
)

type AgentMetrics struct {
	AgentAddress      string `json:"agent_address"`
	PeriodStart     int64  `json:"period_start"`
	PeriodEnd      int64  `json:"period_end"`
	TotalCalls     uint64 `json:"total_calls"`
	SuccessfulCalls uint64 `json:"successful_calls"`
	FailedCalls   uint64 `json:"failed_calls"`
	TotalRevenue  string `json:"total_revenue"`
	AvgLatencyMs  uint64 `json:"avg_latency_ms"`
	UptimePercent uint64 `json:"uptime_percent"`
	LastUpdated  int64  `json:"last_updated"`
}

type NetworkMetrics struct {
	TotalAgents     uint64 `json:"total_agents"`
	ActiveAgents   uint64 `json:"active_agents"`
	TotalServices  uint64 `json:"total_services"`
	ActiveTasks   uint64 `json:"active_tasks"`
	TotalEscrows  uint64 `json:"total_escrows"`
	Volume24h    string `json:"volume_24h"`
	Burn24h      string `json:"burn_24h"`
	AvgReputation uint64 `json:"avg_reputation"`
	TotalStake   string `json:"total_stake"`
	BlockHeight  int64  `json:"block_height"`
	Timestamp   int64  `json:"timestamp"`
}