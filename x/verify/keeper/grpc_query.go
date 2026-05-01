package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/cognize/axon/x/verify/types"
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

func (k queryServer) Verify(goCtx context.Context, req *types.QueryVerifyRequest) (*types.QueryVerifyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	verify, found := k.GetVerify(ctx, req.Address)
	if !found {
		return nil, status.Error(codes.NotFound, "verify not found")
	}

	return &types.QueryVerifyResponse{Verify: &verify}, nil
}

const maxVerifysPerQuery = 200

func (k queryServer) Verifys(goCtx context.Context, req *types.QueryVerifysRequest) (*types.QueryVerifysResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	var verifys []types.Verify
	k.IterateVerifys(ctx, func(verify types.Verify) bool {
		verifys = append(verifys, verify)
		return len(verifys) >= maxVerifysPerQuery
	})
	return &types.QueryVerifysResponse{Verifys: verifys}, nil
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
