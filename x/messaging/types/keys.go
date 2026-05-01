package types

import (
	"encoding/binary"
	"strconv"
)

const (
	ModuleName = "messaging"
	StoreKey   = ModuleName
	RouterKey  = ModuleName

	MaxServiceNameLength     = 64
	MaxServiceDescriptionLen = 256
	MaxTaskTitleLength    = 64
	MaxTaskDescriptionLen = 512
	MaxProposalLength   = 1024
	MaxToolNameLength    = 64
	MaxToolDescriptionLen = 512

	ParamsKey = "Params"

	MessagingKeyPrefix            = "Messaging/value/"
	MessagingCountKey             = "Messaging/count"
	DeregisterQueueKeyPrefix  = "Deregister/queue/"
	ChallengeKeyPrefix        = "Challenge/value/"
	ChallengePoolKeyPrefix    = "Challenge/pool/"
	AIResponseKeyPrefix       = "AIResponse/value/"
	ContributionKeyPrefix     = "Contribution/value/"
	ContributionPoolKey       = "ContributionPool"
	EpochActivityKeyPrefix    = "EpochActivity/"
	AIBonusKeyPrefix          = "AIBonus/"
	RewardPoolKey             = "RewardPool"
	DeployCountKeyPrefix      = "DeployCount/"
	ContractCallKeyPrefix     = "ContractCall/"
	DailyRegKeyPrefix         = "DailyReg/"
	EvidenceTxKeyPrefix       = "EvidenceTx/"
	EvidenceTxHeightKeyPrefix = "EvidenceTxHeight/"

	TotalBlockRewardsMintedKey = "TotalBlockRewardsMinted"
	TotalContributionMintedKey = "TotalContributionMinted"

	PendingReduceStakeKeyPrefix = "PendingReduceStake/"
	LastDailyRegCleanupDayKey   = "LastDailyRegCleanupDay"

	ServiceIdPrefix       = "Service/"
	ServiceCallPrefix    = "ServiceCall/"
	ToolPrefix           = "Tool/"
	ToolCallPrefix       = "ToolCall/"
	GovernanceProposalID = "Governance/next_id"
	GovernanceProposalKeyPrefix = "Governance/proposal/"
	GovernanceVoteKeyPrefix = "Governance/vote/"

	EscrowPrefix       = "Escrow/"
	StablecoinVaultPrefix = "StablecoinVault/"
)

func KeyPendingReduceStake(address string) []byte {
	return []byte(PendingReduceStakeKeyPrefix + address)
}

func KeyMessaging(address string) []byte {
	return []byte(MessagingKeyPrefix + address)
}

func KeyDeregisterQueue(address string) []byte {
	return []byte(DeregisterQueueKeyPrefix + address)
}

func KeyChallenge(epoch uint64) []byte {
	return append([]byte(ChallengeKeyPrefix), Uint64ToBytes(epoch)...)
}

func KeyChallengePool(index uint64) []byte {
	return append([]byte(ChallengePoolKeyPrefix), Uint64ToBytes(index)...)
}

func KeyAIResponse(epoch uint64, validator string) []byte {
	key := []byte(AIResponseKeyPrefix)
	key = append(key, Uint64ToBytes(epoch)...)
	key = append(key, []byte("/"+validator)...)
	return key
}

func KeyAIResponsePrefix(epoch uint64) []byte {
	key := []byte(AIResponseKeyPrefix)
	key = append(key, Uint64ToBytes(epoch)...)
	key = append(key, '/')
	return key
}

func KeyEpochActivity(epoch uint64, address string) []byte {
	key := []byte(EpochActivityKeyPrefix)
	key = append(key, Uint64ToBytes(epoch)...)
	key = append(key, []byte("/"+address)...)
	return key
}

func KeyAIBonus(address string) []byte {
	return []byte(AIBonusKeyPrefix + address)
}

func KeyDeployCount(epoch uint64, address string) []byte {
	key := []byte(DeployCountKeyPrefix)
	key = append(key, Uint64ToBytes(epoch)...)
	key = append(key, []byte("/"+address)...)
	return key
}

func KeyContractCall(epoch uint64, address string) []byte {
	key := []byte(ContractCallKeyPrefix)
	key = append(key, Uint64ToBytes(epoch)...)
	key = append(key, []byte("/"+address)...)
	return key
}

func Uint64ToBytes(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func BytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

func KeyService(id string) []byte {
	return []byte(ServiceIdPrefix + id)
}

func KeyTool(id string) []byte {
	return []byte(ToolPrefix + id)
}

func KeyGovernanceProposal(id uint64) []byte {
	return []byte(GovernanceProposalKeyPrefix + strconv.FormatUint(id, 10))
}

func KeyGovernanceVote(id uint64, voter string) []byte {
	return []byte(GovernanceVoteKeyPrefix + strconv.FormatUint(id, 10) + "/" + voter)
}

func KeyEscrow(id string) []byte {
	return []byte(EscrowPrefix + id)
}

func KeyStablecoinVault(denom string) []byte {
	return []byte(StablecoinVaultPrefix + denom)
}