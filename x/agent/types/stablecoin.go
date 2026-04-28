package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrStablecoinNotFound  = errors.Register(ModuleName, 1300, "stablecoin pool not found")
	ErrStablecoinPaused   = errors.Register(ModuleName, 1301, "stablecoin pool is paused")
	ErrInvalidPrice    = errors.Register(ModuleName, 1302, "invalid price feed")
	ErrOracleOffline  = errors.Register(ModuleName, 1303, "oracle price feed offline")
	ErrInsufficientLiquidity = errors.Register(ModuleName, 1304, "insufficient liquidity")
	ErrDepositTooLow  = errors.Register(ModuleName, 1305, "deposit too low")
	ErrWithdrawalExceedsBalance = errors.Register(ModuleName, 1306, "withdrawal exceeds balance")
	ErrPriceDeviationTooHigh = errors.Register(ModuleName, 1307, "price deviation too high")
)

type Stablecoin struct {
	Denom            string `json:"denom"`
	Name            string `json:"name"`
	Symbol          string `json:"symbol"`
	TotalDeposited   string `json:"total_deposited"`
	TotalWithdrawn  string `json:"total_withdrawn"`
	CurrentSupply  string `json:"current_supply"`
	OraclePrice    string `json:"oracle_price"`
	OracleAddress  string `json:"oracle_address"`
	LastPriceUpdate int64  `json:"last_price_update"`
	MinDeposit     string `json:"min_deposit"`
	FeeBps        uint64 `json:"fee_bps"`
	IsActive       bool   `json:"is_active"`
	LastUpdate   int64  `json:"last_update"`
}

type StablecoinDeposit struct {
	Depositor    string `json:"depositor"`
	Amount     string `json:"amount"`
	LockedCognize string `json:"locked_cognize"`
	DepositTime int64  `json:"deposit_time"`
	Withdrawn  bool   `json:"withdrawn"`
}

type StablecoinSwap struct {
	SwapID        string `json:"swap_id"`
	Depositor     string `json:"depositor"`
	CognizeAmount  string `json:"cognize_amount"`
	StableAmount string `json:"stable_amount"`
	Rate         string `json:"rate"`
	Direction    string `json:"direction"`
	Status      string `json:"status"`
	BlockHeight  int64  `json:"block_height"`
	BlockTime   int64  `json:"block_time"`
}

type PriceFeed struct {
	Price     string `json:"price"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature"`
}

const (
	StablecoinCUSD    = "cusd"
	StablecoinCEUR    = "ceur"
	StablecoinCJPY    = "cjypt"
	MaxPriceDeviation = 500
	OracleTimeout    = 300
)

func KeyStablecoin(denom string) []byte {
	return []byte("stablecoin/" + denom)
}

func KeyStablecoinDeposit(denom, depositor string) []byte {
	return []byte("stablecoin/deposit/" + denom + "/" + depositor)
}

func KeyStablecoinSwap(swapID string) []byte {
	return []byte("stablecoin/swap/" + swapID)
}

type DynamicParams struct {
	NetworkImportance uint64 `json:"network_importance"`
	TotalStake        string `json:"total_stake"`
	ActiveAgents      uint64 `json:"active_agents"`
	Transactions24h  uint64 `json:"transactions_24h"`
	GasUsage24h       uint64 `json:"gas_usage_24h"`
	MinStake         uint64 `json:"min_stake"`
	MinStakeDynamic  uint64 `json:"min_stake_dynamic"`
	RegisterBurn    uint64 `json:"register_burn"`
	DeployBurn      uint64 `json:"deploy_burn"`
	EpochLength     uint64 `json:"epoch_length"`
	LastUpdate     int64  `json:"last_update"`
}

type InsurancePool struct {
	PoolID      string `json:"pool_id"`
	Agent      string `json:"agent"`
	CoverAmount string `json:"cover_amount"`
	Premium    string `json:"premium"`
	StartBlock int64  `json:"start_block"`
	EndBlock  int64  `json:"end_block"`
	Active    bool   `json:"active"`
	ClaimsPaid string `json:"claims_paid"`
	Status    string `json:"status"`
}

type InsuranceClaim struct {
	ClaimID     string `json:"claim_id"`
	PoolID     string `json:"pool_id"`
	Agent      string `json:"agent"`
	Amount     string `json:"amount"`
	Reason     string `json:"reason"`
	BlockTime  int64  `json:"block_time"`
	Status    string `json:"status"`
	ApprovedBy string `json:"approved_by"`
	PaidAt    int64  `json:"paid_at"`
}

const (
	InsuranceStatusActive  = "active"
	InsuranceStatusExpired = "expired"
	InsuranceStatusClaimed = "claimed"
)

func KeyInsurancePool(agent string) []byte {
	return []byte("insurance/pool/" + agent)
}

func KeyInsuranceClaim(claimID string) []byte {
	return []byte("insurance/claim/" + claimID)
}