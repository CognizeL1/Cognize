package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrIBCChannelNotFound   = errors.Register(ModuleName, 1600, "IBC channel not found")
	ErrIBCChannelOpen   = errors.Register(ModuleName, 1601, "IBC channel already open")
	ErrIBCTimeout    = errors.Register(ModuleName, 1602, "IBC timeout")
	ErrIBCFeeTooLow  = errors.Register(ModuleName, 1603, "IBC fee too low")
	ErrMultiSigNotFound = errors.Register(ModuleName, 1650, "multisig wallet not found")
	ErrMultiSigNotAuthorized = errors.Register(ModuleName, 1651, "not authorized for multisig")
	ErrMultiSigInsufficient = errors.Register(ModuleName, 1652, "insufficient signers")
	ErrDAONotFound    = errors.Register(ModuleName, 1700, "DAO not found")
	ErrDAOAlreadyMember = errors.Register(ModuleName, 1701, "already member")
	ErrStreamingNotFound = errors.Register(ModuleName, 1750, "streaming not found")
	ErrStreamingActive = errors.Register(ModuleName, 1751, "streaming already active")
)

type IBCChannel struct {
	ChannelID  string `json:"channel_id"`
	PortID    string `json:"port_id"`
	CounterChainID string `json:"counter_chain_id"`
	State     string `json:"state"`
	FeeBps    uint64 `json:"fee_bps"`
	MinAmount string `json:"min_amount"`
	MaxAmount string `json:"max_amount"`
	Enabled  bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

type IBCTransfer struct {
	TransferID string `json:"transfer_id"`
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Amount   string `json:"amount"`
	Denom    string `json:"denom"`
	Fee      string `json:"fee"`
	Status   string `json:"status"`
	SourceChain string `json:"source_chain"`
	TargetChain string `json:"target_chain"`
	CreatedAt int64 `json:"created_at"`
	CompletedAt int64 `json:"completed_at"`
}

type MultisigWallet struct {
	WalletID   string   `json:"wallet_id"`
	Name      string   `json:"name"`
	Owners    []string `json:"owners"`
	Threshold uint64   `json:"threshold"`
	CreatedBy string   `json:"created_by"`
	Balance   string   `json:"balance"`
	CreatedAt int64    `json:"created_at"`
	Status    string   `json:"status"`
}

type MultisigTx struct {
	TxID        string `json:"tx_id"`
	WalletID    string `json:"wallet_id"`
	ProposedBy  string `json:"proposed_by"`
	To         string `json:"to"`
	Amount     string `json:"amount"`
	Memo       string `json:"memo"`
	Signatures  []string `json:"signatures"`
	Executed   bool    `json:"executed"`
	ExecutedAt int64   `json:"executed_at"`
	CreatedAt  int64   `json:"created_at"`
}

type DAO struct {
	DAOID     string `json:"dao_id"`
	Name      string `json:"name"`
	Creator   string `json:"creator"`
	Members   []string `json:"members"`
	Token     string `json:"token"`
	Quorum    uint64 `json:"quorum"`
	Threshold uint64 `json:"threshold"`
	Proposals []DAOProposal `json:"proposals"`
	Treasurer string `json:"treasurer"`
	CreatedAt int64  `json:"created_at"`
	Status   string  `json:"status"`
}

type DAOProposal struct {
	ProposalID string `json:"proposal_id"`
	DAOID    string `json:"dao_id"`
	Title    string `json:"title"`
	Action   string `json:"action"`
	Amount   string `json:"amount"`
	Votes    uint64 `json:"votes"`
	Status   string `json:"status"`
	CreatedAt int64 `json:"created_at"`
}

type StreamingPlan struct {
	PlanID      string `json:"plan_id"`
	Sender      string `json:"sender"`
	Recipient  string `json:"recipient"`
	TotalAmount string `json:"total_amount"`
	PerBlock  string `json:"per_block"`
	BlocksRemaining uint64 `json:"blocks_remaining"`
	StartBlock int64  `json:"start_block"`
	EndBlock  int64  `json:"end_block"`
	Active  bool    `json:"active"`
	Paused bool    `json:"paused"`
}

func KeyIBCChannel(chainID string) []byte {
	return []byte("ibc/channel/" + chainID)
}

func KeyIBCTransfer(transferID string) []byte {
	return []byte("ibc/transfer/" + transferID)
}

func KeyMultisigWallet(walletID string) []byte {
	return []byte("multisig/wallet/" + walletID)
}

func KeyMultisigTx(txID string) []byte {
	return []byte("multisig/tx/" + txID)
}

func KeyDAO(daoID string) []byte {
	return []byte("dao/" + daoID)
}

func KeyStreamingPlan(planID string) []byte {
	return []byte("streaming/plan/" + planID)
}