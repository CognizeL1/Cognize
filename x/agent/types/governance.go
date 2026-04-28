package types

import (
	"fmt"

	"cosmossdk.io/errors"
)

var (
	ErrProposalNotFound      = errors.Register(ModuleName, 2, "proposal not found")
	ErrVotingPeriodEnded = errors.Register(ModuleName, 3, "voting period has ended")
	ErrVotingPeriodActive = errors.Register(ModuleName, 4, "voting period is still active")
	ErrQuorumNotReached  = errors.Register(ModuleName, 5, "quorum not reached")
	ErrProposalRejected = errors.Register(ModuleName, 6, "proposal rejected")
	ErrProposalVetoed  = errors.Register(ModuleName, 7, "proposal vetoed")
	ErrAlreadyVoted     = errors.Register(ModuleName, 8, "agent already voted")
	ErrInvalidProposalType = errors.Register(ModuleName, 9, "invalid proposal type")
)

type GovernanceProposal struct {
	ProposalID     uint64 `json:"proposal_id"`
	Title          string `json:"title"`
	Description   string `json:"description"`
	ProposalType  string `json:"proposal_type"`
	Author         string `json:"author"`
	Deposits       string `json:"deposits"`
	VoteStartBlock int64  `json:"vote_start_block"`
	VoteEndBlock   int64  `json:"vote_end_block"`
	Status         string `json:"status"`
	ForVotes       string `json:"for_votes"`
	AgainstVotes   string `json:"against_votes"`
	VetoVotes      string `json:"veto_votes"`
	TotalVoters    uint64 `json:"total_voters"`
	Executed       bool   `json:"executed"`
	ExecutedBlock  int64  `json:"executed_block"`
	CreatedAt      int64  `json:"created_at"`
}

func (p GovernanceProposal) String() string {
	return fmt.Sprintf("Proposal %d: %s (%s)", p.ProposalID, p.Title, p.Status)
}

func (p GovernanceProposal) Validate() error {
	if p.Title == "" {
		return fmt.Errorf("title is required")
	}
	if p.Description == "" {
		return fmt.Errorf("description is required")
	}
	validTypes := map[string]bool{
		"parameter": true,
		"treasury":  true,
		"upgrade":   true,
		"emergency": true,
		"community": true,
	}
	if !validTypes[p.ProposalType] {
		return ErrInvalidProposalType
	}
	return nil
}

type Vote struct {
	ProposalID uint64 `json:"proposal_id"`
	Voter      string `json:"voter"`
	VoteType   string `json:"vote_type"`
	Reasoning string `json:"reasoning"`
	Weight    string `json:"weight"`
	Block     int64  `json:"block"`
	Timestamp int64 `json:"timestamp"`
}

func KeyProposal(id uint64) []byte {
	return []byte(fmt.Sprintf("proposal/%050d", id))
}

func KeyVote(proposalID uint64, voter string) []byte {
	return []byte(fmt.Sprintf("vote/%050d/%s", proposalID, voter))
}