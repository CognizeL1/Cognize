package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/oracle/types"
)

type msgServer struct {
	types.UnimplementedMsgServer
	Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) Register(goCtx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	if k.IsOracle(ctx, msg.Sender) {
		return nil, types.ErrOracleAlreadyRegistered
	}

	if msg.Stake.Denom != "aaxon" {
		return nil, fmt.Errorf("invalid stake denom: expected aaxon, got %s", msg.Stake.Denom)
	}
	minStakeInt := sdkmath.NewIntFromBigInt(new(big.Int).Mul(big.NewInt(int64(params.MinRegisterStake)), oneAxon))
	minStake := sdk.NewCoin("aaxon", minStakeInt)
	if msg.Stake.IsLT(minStake) {
		return nil, types.ErrInsufficientStake
	}

	// Per-address daily registration limit (whitepaper §10.5)
	if k.GetDailyRegisterCount(ctx, msg.Sender) >= types.MaxDailyRegistrations {
		return nil, types.ErrDailyRegisterLimitExceeded
	}

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	stakeCoins := sdk.NewCoins(msg.Stake)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, stakeCoins); err != nil {
		return nil, err
	}

	burnInt := sdkmath.NewIntFromBigInt(new(big.Int).Mul(big.NewInt(int64(params.RegisterBurnAmount)), oneAxon))
	burnAmount := sdk.NewCoin("aaxon", burnInt)
	burnCoins := sdk.NewCoins(burnAmount)
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, burnCoins); err != nil {
		return nil, err
	}

	if len(msg.Capabilities) > 1024 {
		return nil, fmt.Errorf("capabilities too long: max 1024 bytes")
	}
	if len(msg.Model) > 256 {
		return nil, fmt.Errorf("model name too long: max 256 bytes")
	}
	capabilities := strings.Split(msg.Capabilities, ",")
	for i := range capabilities {
		capabilities[i] = strings.TrimSpace(capabilities[i])
	}

	oracle := types.Oracle{
		Address:          msg.Sender,
		OracleId:          generateOracleID(msg.Sender, ctx.BlockHeight()),
		Capabilities:     capabilities,
		Model:            msg.Model,
		Reputation:       params.InitialReputation,
		Status:           types.OracleStatus_ORACLE_STATUS_ONLINE,
		StakeAmount:      msg.Stake,
		BurnedAtRegister: burnAmount,
		RegisteredAt:     ctx.BlockHeight(),
		LastHeartbeat:    ctx.BlockHeight(),
	}

	k.SetOracle(ctx, oracle)
	k.BootstrapLegacyReputation(ctx, oracle.Address, oracle.Reputation)
	k.IncrementDailyRegisterCount(ctx, msg.Sender)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"oracle_registered",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("oracle_id", oracle.OracleId),
		sdk.NewAttribute("stake", msg.Stake.String()),
		sdk.NewAttribute("burned", burnAmount.String()),
		sdk.NewAttribute("reputation", fmt.Sprintf("%d", oracle.Reputation)),
	))

	return &types.MsgRegisterResponse{OracleId: oracle.OracleId}, nil
}

func (k msgServer) AddStake(goCtx context.Context, msg *types.MsgAddStake) (*types.MsgAddStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	return k.Keeper.AddStakeToOracle(ctx, msg.Sender, msg.Stake, senderAddr)
}

func (k msgServer) UpdateOracle(goCtx context.Context, msg *types.MsgUpdateOracle) (*types.MsgUpdateOracleResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	oracle, found := k.GetOracle(ctx, msg.Sender)
	if !found {
		return nil, types.ErrOracleNotFound
	}

	if oracle.Status == types.OracleStatus_ORACLE_STATUS_SUSPENDED {
		return nil, types.ErrOracleSuspended
	}

	if len(msg.Capabilities) > 1024 {
		return nil, fmt.Errorf("capabilities too long: max 1024 bytes")
	}
	if len(msg.Model) > 256 {
		return nil, fmt.Errorf("model name too long: max 256 bytes")
	}
	if msg.Capabilities != "" {
		caps := strings.Split(msg.Capabilities, ",")
		for i := range caps {
			caps[i] = strings.TrimSpace(caps[i])
		}
		oracle.Capabilities = caps
	}
	if msg.Model != "" {
		oracle.Model = msg.Model
	}

	k.SetOracle(ctx, oracle)
	return &types.MsgUpdateOracleResponse{}, nil
}

func (k msgServer) Heartbeat(goCtx context.Context, msg *types.MsgHeartbeat) (*types.MsgHeartbeatResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	oracle, found := k.GetOracle(ctx, msg.Sender)
	if !found {
		return nil, types.ErrOracleNotFound
	}

	if oracle.Status == types.OracleStatus_ORACLE_STATUS_SUSPENDED {
		return nil, types.ErrOracleSuspended
	}

	if ctx.BlockHeight()-oracle.LastHeartbeat < params.HeartbeatInterval {
		return nil, types.ErrHeartbeatTooFrequent
	}

	oracle.LastHeartbeat = ctx.BlockHeight()
	oracle.Status = types.OracleStatus_ORACLE_STATUS_ONLINE
	k.SetOracle(ctx, oracle)

	k.IncrementEpochActivity(ctx, msg.Sender)

	return &types.MsgHeartbeatResponse{}, nil
}

func (k msgServer) Deregister(goCtx context.Context, msg *types.MsgDeregister) (*types.MsgDeregisterResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	oracle, found := k.GetOracle(ctx, msg.Sender)
	if !found {
		return nil, types.ErrOracleNotFound
	}

	if k.HasDeregisterRequest(ctx, msg.Sender) {
		return nil, types.ErrDeregisterAlreadyQueued
	}

	k.SetDeregisterRequest(ctx, msg.Sender, ctx.BlockHeight())

	oracle.Status = types.OracleStatus_ORACLE_STATUS_SUSPENDED
	k.SetOracle(ctx, oracle)

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"oracle_deregister_requested",
		sdk.NewAttribute("address", msg.Sender),
		sdk.NewAttribute("cooldown_blocks", fmt.Sprintf("%d", types.DeregisterCooldownBlocks)),
		sdk.NewAttribute("refund_at_block", fmt.Sprintf("%d", ctx.BlockHeight()+types.DeregisterCooldownBlocks)),
	))

	return &types.MsgDeregisterResponse{}, nil
}

func (k msgServer) ReduceStake(goCtx context.Context, msg *types.MsgReduceStake) (*types.MsgReduceStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.Keeper.ReduceStakeFromOracle(ctx, msg.Sender, msg.Amount); err != nil {
		return nil, err
	}
	return &types.MsgReduceStakeResponse{}, nil
}

func (k msgServer) ClaimReducedStake(goCtx context.Context, msg *types.MsgClaimReducedStake) (*types.MsgClaimReducedStakeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.Keeper.ClaimReducedStake(ctx, msg.Sender); err != nil {
		return nil, err
	}
	return &types.MsgClaimReducedStakeResponse{}, nil
}

func (k msgServer) SubmitAIChallengeResponse(goCtx context.Context, msg *types.MsgSubmitAIChallengeResponse) (*types.MsgSubmitAIChallengeResponseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	oracle, found := k.GetOracle(ctx, msg.Sender)
	if !found {
		return nil, types.ErrOracleNotFound
	}
	if oracle.Status == types.OracleStatus_ORACLE_STATUS_SUSPENDED {
		return nil, types.ErrOracleSuspended
	}
	if !k.isActiveValidatorAddress(ctx, msg.Sender) {
		return nil, types.ErrValidatorRequired
	}

	challenge, found := k.GetChallenge(ctx, msg.Epoch)
	if !found {
		return nil, types.ErrChallengeNotActive
	}

	if ctx.BlockHeight() > challenge.DeadlineBlock {
		return nil, types.ErrChallengeWindowClosed
	}

	store := ctx.KVStore(k.storeKey)
	key := types.KeyAIResponse(msg.Epoch, msg.Sender)
	if store.Has(key) {
		return nil, types.ErrAlreadySubmitted
	}

	response := types.AIResponse{
		ValidatorAddress: msg.Sender,
		Epoch:            msg.Epoch,
		CommitHash:       msg.CommitHash,
		Evaluated:        false,
	}

	bz := k.cdc.MustMarshal(&response)
	store.Set(key, bz)

	k.IncrementEpochActivity(ctx, msg.Sender)

	return &types.MsgSubmitAIChallengeResponseResponse{}, nil
}

func (k msgServer) RevealAIChallengeResponse(goCtx context.Context, msg *types.MsgRevealAIChallengeResponse) (*types.MsgRevealAIChallengeResponseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	if _, found := k.GetOracle(ctx, msg.Sender); !found {
		return nil, types.ErrOracleNotFound
	}
	if k.HasDeregisterRequest(ctx, msg.Sender) {
		return nil, types.ErrDeregisterCooldown
	}
	if !k.isActiveValidatorAddress(ctx, msg.Sender) {
		return nil, types.ErrValidatorRequired
	}

	challenge, found := k.GetChallenge(ctx, msg.Epoch)
	if !found {
		return nil, types.ErrChallengeNotActive
	}

	// Reveal must happen after commit deadline
	if ctx.BlockHeight() <= challenge.DeadlineBlock {
		return nil, types.ErrRevealTooEarly
	}

	// Reveal must happen within the reveal window
	revealDeadline := challenge.DeadlineBlock + params.AiChallengeWindow
	if ctx.BlockHeight() > revealDeadline {
		return nil, types.ErrRevealWindowClosed
	}

	store := ctx.KVStore(k.storeKey)
	key := types.KeyAIResponse(msg.Epoch, msg.Sender)
	bz := store.Get(key)
	if bz == nil {
		return nil, types.ErrChallengeNotActive
	}

	var response types.AIResponse
	k.cdc.MustUnmarshal(bz, &response)

	if len(msg.RevealData) > 512 {
		return nil, fmt.Errorf("reveal data too long: max 512 bytes")
	}

	if response.Evaluated {
		return nil, types.ErrAlreadyEvaluated
	}

	// Commit format: SHA256(sender + ":" + revealData) — address acts as implicit salt
	commitInput := msg.Sender + ":" + msg.RevealData
	revealHash := sha256.Sum256([]byte(commitInput))
	if hex.EncodeToString(revealHash[:]) != response.CommitHash {
		return nil, types.ErrInvalidReveal
	}

	response.RevealData = msg.RevealData
	bz = k.cdc.MustMarshal(&response)
	store.Set(key, bz)

	return &types.MsgRevealAIChallengeResponseResponse{}, nil
}

func generateOracleID(address string, blockHeight int64) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", address, blockHeight)))
	return fmt.Sprintf("oracle-%s", hex.EncodeToString(hash[:8]))
}
