package types

import (
	"fmt"

	"cosmossdk.io/errors"
)

var (
	ErrPrivacyKeyNotFound     = errors.Register(ModuleName, 1500, "privacy key not found")
	ErrPrivacyKeyExpired    = errors.Register(ModuleName, 1501, "privacy key expired")
	ErrPrivacyKeyRevoked  = errors.Register(ModuleName, 1502, "privacy key revoked")
	ErrServicePrivate    = errors.Register(ModuleName, 1503, "service requires privacy access")
	ErrInvalidKey      = errors.Register(ModuleName, 1504, "invalid privacy key")
	ErrKeyAlreadyUsed   = errors.Register(ModuleName, 1505, "privacy key already used")
	ErrMixNotFound    = errors.Register(ModuleName, 1506, "mix not found")
	ErrMixInProgress  = errors.Register(ModuleName, 1507, "mix already in progress")
	ErrInvalidDenomination = errors.Register(ModuleName, 1508, "invalid denomination")
	ErrMixPhaseWrong   = errors.Register(ModuleName, 1509, "mix phase not active")
	ErrCommitmentNotFound = errors.Register(ModuleName, 1510, "commitment not found")
	ErrDoubleSpend    = errors.Register(ModuleName, 1511, "double spend detected")
	ErrTooManyRequests = errors.Register(ModuleName, 1512, "rate limit exceeded")
)

type PrivacyKey struct {
	KeyID        string `json:"key_id"`
	Issuer       string `json:"issuer"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Key          string `json:"key"`
	AccessLevel  string `json:"access_level"`
	MaxUses     uint64 `json:"max_uses"`
	UsedCount   uint64 `json:"used_count"`
	ExpiresAt   int64  `json:"expires_at"`
	Revoked     bool   `json:"revoked"`
	CreatedAt   int64  `json:"created_at"`
	Metadata    string `json:"metadata"`
}

type PrivacyKeyUsage struct {
	KeyID        string `json:"key_id"`
	User        string `json:"user"`
	UsedAt      int64  `json:"used_at"`
	BlockHeight int64  `json:"block_height"`
	IPHash     string `json:"ip_hash"`
}

type PrivateService struct {
	ServiceID      string `json:"service_id"`
	AgentAddress  string `json:"agent_address"`
	Name         string `json:"name"`
	IsPrivate    bool   `json:"is_private"`
	AccessType   string `json:"access_type"`
	Price       string `json:"price"`
	AccessKeyPrice string `json:"access_key_price"`
	AllowedList  string `json:"allowed_list"`
	MaxAccess  uint64 `json:"max_access"`
	CurrentAccess uint64 `json:"current_access"`
	Status      string `json:"status"`
}

type PrivacyPool struct {
	PoolID          string `json:"pool_id"`
	Denomination    string `json:"denomination"`
	TotalDeposited string `json:"total_deposited"`
	TotalWithdrawn string `json:"total_withdrawn"`
	Status      string `json:"status"`
	Epoch        uint64 `json:"epoch"`
	Participants uint64 `json:"participants"`
	FeeBps       uint64 `json:"fee_bps"`
	MinDeposit    string `json:"min_deposit"`
	StartBlock   int64  `json:"start_block"`
	EndBlock    int64  `json:"end_block"`
}

type MixCommitment struct {
	Commitment string `json:"commitment"`
	Hash       string `json:"hash"`
	Deposit    string `json:"deposit"`
	Depositor  string `json:"depositor"`
	Status    string `json:"status"`
	Block     int64  `json:"block"`
	LeafIndex uint64 `json:"leaf_index"`
}

type MixWithdrawal struct {
	WithdrawalID  string `json:"withdrawal_id"`
	Root          string `json:"root"`
	Proof         string `json:"proof"`
	Recipient    string `json:"recipient"`
	Amount       string `json:"amount"`
	Status       string `json:"status"`
	Block         int64  `json:"block"`
	Fee           string `json:"fee"`
}

type PrivacyTier struct {
	TierName     string `json:"tier_name"`
	RequiredStake string `json:"required_stake"`
	RequiredRep  uint64 `json:"required_rep"`
	MaxMixSize   uint64 `json:"max_mix_size"`
	Priority    uint64 `json:"priority"`
	FeeDiscount uint64 `json:"fee_discount"`
}

const (
	AccessLevelPublic        = "public"
	AccessLevelPrivate   = "private"
	AccessLevelTokenGated = "token_gated"
	AccessLevelWhitelist = "whitelist"

	MixPhaseCommit  = "commit"
	MixPhaseWithdraw = "withdraw"
	MixPhaseDone   = "done"

	PrivacyTierBasic     = "basic"
	PrivacyTierSilver   = "silver"
	PrivacyTierGold   = "gold"
	PrivacyTierDiamond = "diamond"
)

func KeyPrivacyKey(keyID string) []byte {
	return []byte("privacy/key/" + keyID)
}

func KeyPrivacyKeyByResource(resourceType, resourceID string) []byte {
	return []byte("privacy/resource/" + resourceType + "/" + resourceID)
}

func KeyPrivateService(serviceID string) []byte {
	return []byte("privacy/service/" + serviceID)
}

func KeyMixCommitment(poolID string, commitment string) []byte {
	return []byte("privacy/mix/" + poolID + "/" + commitment)
}

func KeyMixWithdrawal(withdrawalID string) []byte {
	return []byte("privacy/withdraw/" + withdrawalID)
}

func KeyPrivacyPool(poolID string) []byte {
	return []byte("privacy/pool/" + poolID)
}

func KeyPrivacyTier(tier string) []byte {
	return []byte("privacy/tier/" + tier)
}

func GeneratePrivacyKey(issuer, resourceType, resourceID string, nonce uint64) string {
	return fmt.Sprintf("pkey-%s-%s-%s-%d", issuer[:8], resourceType, resourceID[:8], nonce)
}