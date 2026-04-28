package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	ProposalMinStake     = 10
	ProposalMinReputation = 20
	ProposalVotingPeriod = 720
	QuorumPercentage   = 3340
	PassPercentage     = 5000
	VetoThreshold       = 3340
)

type ProposalJSON struct {
	ProposalID     uint64 `json:"proposal_id"`
	Title          string `json:"title"`
	Description    string `json:"description"`
	ProposalType   string `json:"proposal_type"`
	Author         string `json:"author"`
	Deposits       string `json:"deposits"`
	VoteStart      int64  `json:"vote_start"`
	VoteEnd        int64  `json:"vote_end"`
	Status         string `json:"status"`
	ForVotes       string `json:"for_votes"`
	AgainstVotes   string `json:"against_votes"`
	VetoVotes      string `json:"veto_votes"`
	TotalVoters   uint64 `json:"total_voters"`
	Executed       bool   `json:"executed"`
	ExecutedBlock  int64  `json:"executed_block"`
	CreatedAt      int64  `json:"created_at"`
}

func (k Keeper) SubmitProposal(ctx sdk.Context, title, desc, propType string, author string) error {
	if title == "" || desc == "" {
		return fmt.Errorf("title and description required")
	}
	validTypes := map[string]bool{
		"parameter": true, "treasury": true, "upgrade": true,
		"emergency": true, "community": true,
	}
	if !validTypes[propType] {
		return types.ErrInvalidProposalType
	}

	authorAgent, found := k.GetAgent(ctx, author)
	if !found {
		return types.ErrAgentNotFound
	}

	stakeAmount := authorAgent.StakeAmount.Amount
	if stakeAmount.LT(math.NewInt(ProposalMinStake)) {
		return types.ErrInsufficientStake
	}

	if authorAgent.Reputation < ProposalMinReputation {
		return types.ErrReputationTooLow
	}

	store := ctx.KVStore(k.storeKey)
	proposalID := k.getNextProposalID(store)

	proposal := ProposalJSON{
		ProposalID:    proposalID,
		Title:         title,
		Description:   desc,
		ProposalType: propType,
		Author:        author,
		Deposits:      "0",
		VoteStart:     ctx.BlockHeight(),
		VoteEnd:       ctx.BlockHeight() + ProposalVotingPeriod,
		Status:        "voting",
		ForVotes:      "0",
		AgainstVotes: "0",
		VetoVotes:     "0",
		TotalVoters:   0,
		Executed:      false,
		CreatedAt:     ctx.BlockTime().Unix(),
	}

	bz, err := json.Marshal(&proposal)
	if err != nil {
		return err
	}
	store.Set(types.KeyGovernanceProposal(proposalID), bz)

	return nil
}

func (k Keeper) CastVote(ctx sdk.Context, proposalID uint64, voter string, voteType string, reasoning string) error {
	validVotes := map[string]bool{"for": true, "against": true, "veto": true}
	if !validVotes[voteType] {
		return fmt.Errorf("invalid vote type")
	}

	voterAgent, found := k.GetAgent(ctx, voter)
	if !found {
		return types.ErrAgentNotFound
	}

	stakeAmount := voterAgent.StakeAmount.Amount
	if stakeAmount.LT(math.NewInt(ProposalMinStake)) {
		return types.ErrInsufficientStake
	}

	if voterAgent.Reputation < 10 {
		return types.ErrReputationTooLow
	}

	store := ctx.KVStore(k.storeKey)
	proposalKey := types.KeyGovernanceProposal(proposalID)
	bz := store.Get(proposalKey)
	if bz == nil {
		return types.ErrProposalNotFound
	}

	var proposal ProposalJSON
	if err := json.Unmarshal(bz, &proposal); err != nil {
		return err
	}

	if ctx.BlockHeight() > proposal.VoteEnd {
		return types.ErrVotingPeriodEnded
	}

	voteKey := types.KeyGovernanceVote(proposalID, voter)
	if store.Get(voteKey) != nil {
		return types.ErrAlreadyVoted
	}

	weight := calculateVotingWeight(stakeAmount, voterAgent.Reputation)

	vote := types.Vote{
		ProposalID: proposalID,
		Voter:       voter,
		VoteType:    voteType,
		Reasoning:   reasoning,
		Weight:      weight.String(),
		Block:       ctx.BlockHeight(),
		Timestamp:   ctx.BlockTime().Unix(),
	}

	bz, _ = json.Marshal(&vote)
	store.Set(voteKey, bz)

	forVotes, _ := math.NewIntFromString(proposal.ForVotes)
	againstVotes, _ := math.NewIntFromString(proposal.AgainstVotes)
	vetoVotes, _ := math.NewIntFromString(proposal.VetoVotes)

	switch voteType {
	case "for":
		forVotes = forVotes.Add(weight)
	case "against":
		againstVotes = againstVotes.Add(weight)
	case "veto":
		vetoVotes = vetoVotes.Add(weight)
	}
	proposal.TotalVoters++
	proposal.ForVotes = forVotes.String()
	proposal.AgainstVotes = againstVotes.String()
	proposal.VetoVotes = vetoVotes.String()

	bz, _ = json.Marshal(&proposal)
	store.Set(proposalKey, bz)

	return nil
}

func (k Keeper) ExecuteProposalIfPassed(ctx sdk.Context, proposalID uint64) error {
	store := ctx.KVStore(k.storeKey)
	proposalKey := types.KeyGovernanceProposal(proposalID)
	bz := store.Get(proposalKey)
	if bz == nil {
		return types.ErrProposalNotFound
	}

	var proposal ProposalJSON
	if err := json.Unmarshal(bz, &proposal); err != nil {
		return err
	}

	if proposal.Executed {
		return nil
	}

	if ctx.BlockHeight() < proposal.VoteEnd {
		return types.ErrVotingPeriodActive
	}

	forVotes, _ := math.NewIntFromString(proposal.ForVotes)
	againstVotes, _ := math.NewIntFromString(proposal.AgainstVotes)
	vetoVotes, _ := math.NewIntFromString(proposal.VetoVotes)
	totalVotes := forVotes.Add(againstVotes).Add(vetoVotes)

	if totalVotes.IsZero() {
		return types.ErrQuorumNotReached
	}

	forVotesPct := forVotes.Mul(math.NewInt(10000)).Quo(totalVotes)
	vetoVotesPct := vetoVotes.Mul(math.NewInt(10000)).Quo(totalVotes)

	if vetoVotesPct.GTE(math.NewInt(VetoThreshold)) {
		proposal.Status = "vetoed"
		bz, _ = json.Marshal(&proposal)
		store.Set(proposalKey, bz)
		return types.ErrProposalVetoed
	}

	if forVotesPct.LTE(math.NewInt(PassPercentage)) {
		proposal.Status = "rejected"
		bz, _ = json.Marshal(&proposal)
		store.Set(proposalKey, bz)
		return types.ErrProposalRejected
	}

	proposal.Status = "passed"
	proposal.Executed = true
	proposal.ExecutedBlock = ctx.BlockHeight()

	bz, _ = json.Marshal(&proposal)
	store.Set(proposalKey, bz)

	switch proposal.ProposalType {
	case "parameter":
		k.executeParameterChange(ctx, proposal.Description)
	case "treasury":
		k.executeTreasuryAction(ctx, proposal.Description)
	case "upgrade":
		k.executeUpgrade(ctx, proposal.Description)
	}

	return nil
}

func calculateVotingWeight(stake math.Int, reputation uint64) math.Int {
	if stake.IsZero() {
		return math.ZeroInt()
	}
	stakeBig := stake.BigInt()
	stakeWeight := new(big.Float).SetInt(stakeBig)
	stakeWeight.Sqrt(stakeWeight)

	repBase := big.NewFloat(110)
	repWeight := big.NewFloat(float64(reputation) + 10)
	repWeight.Quo(repWeight, repBase)

	weight := new(big.Float).Mul(stakeWeight, repWeight)
	weightInt, _ := math.NewIntFromString(weight.String())

	return weightInt.Mul(math.NewInt(100))
}

func (k Keeper) executeParameterChange(ctx sdk.Context, description string) {
}

func (k Keeper) executeTreasuryAction(ctx sdk.Context, description string) {
}

func (k Keeper) executeUpgrade(ctx sdk.Context, description string) {
}

func (k Keeper) getNextProposalID(store storetypes.KVStore) uint64 {
	bz := store.Get([]byte(types.GovernanceProposalIDKey))
	if bz == nil {
		return 1
	}
	id := types.BytesToUint64(bz)
	nextID := id + 1
	store.Set([]byte(types.GovernanceProposalIDKey), types.Uint64ToBytes(nextID))
	return id
}