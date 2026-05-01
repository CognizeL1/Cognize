package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cognize/axon/x/state/types"
)

type queryServer struct {
	types.UnimplementedQueryServer
	Keeper
}

var _ types.QueryServer = queryServer{}

func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

func (k queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

func (k queryServer) ustate(goCtx context.Context, req *types.QueryustateRequest) (*types.QueryustateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	state, found := k.Getustate(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "state not found")
	}

	return &types.QueryustateResponse{ustate: &state}, nil
}

const maxustatesPerQuery = 200

func (k queryServer) ustates(goCtx context.Context, req *types.QueryustatesRequest) (*types.QueryustatesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	var states []types.ustate
	k.Iterateustates(ctx, func(state types.ustate) bool {
		states = append(states, state)
		return len(states) >= maxustatesPerQuery
	})
	return &types.QueryustatesResponse{ustates: states}, nil
}

func (k queryServer) Reputation(goCtx context.Context, req *types.QueryReputationRequest) (*types.QueryReputationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	rep := k.GetReputation(ctx, req.Address)
	return &types.QueryReputationResponse{Reputation: rep}, nil
}

func (k queryServer) CurrentChallenge(goCtx context.Context, req *types.QueryCurrentChallengeRequest) (*types.QueryCurrentChallengeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	epoch := k.GetCurrentEpoch(ctx)
	challenge, found := k.GetChallenge(ctx, epoch)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no active challenge for epoch %d", epoch)
	}

	return &types.QueryCurrentChallengeResponse{Challenge: &challenge}, nil
}
