package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cognize/axon/x/capabilities/types"
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

func (k queryServer) ucapabilities(goCtx context.Context, req *types.QueryucapabilitiesRequest) (*types.QueryucapabilitiesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	capabilities, found := k.Getucapabilities(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "capabilities not found")
	}

	return &types.QueryucapabilitiesResponse{ucapabilities: &capabilities}, nil
}

const maxucapabilitiessPerQuery = 200

func (k queryServer) ucapabilitiess(goCtx context.Context, req *types.QueryucapabilitiessRequest) (*types.QueryucapabilitiessResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	var capabilitiess []types.ucapabilities
	k.Iterateucapabilitiess(ctx, func(capabilities types.ucapabilities) bool {
		capabilitiess = append(capabilitiess, capabilities)
		return len(capabilitiess) >= maxucapabilitiessPerQuery
	})
	return &types.QueryucapabilitiessResponse{ucapabilitiess: capabilitiess}, nil
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
