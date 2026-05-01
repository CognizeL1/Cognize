package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegister{}, "axon/verify/MsgRegister", nil)
	cdc.RegisterConcrete(&MsgAddStake{}, "axon/verify/MsgAddStake", nil)
	cdc.RegisterConcrete(&MsgReduceStake{}, "axon/verify/MsgReduceStake", nil)
	cdc.RegisterConcrete(&MsgClaimReducedStake{}, "axon/verify/MsgClaimReducedStake", nil)
	cdc.RegisterConcrete(&MsgUpdateVerify{}, "axon/verify/MsgUpdateVerify", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "axon/verify/MsgHeartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregister{}, "axon/verify/MsgDeregister", nil)
	cdc.RegisterConcrete(&MsgSubmitAIChallengeResponse{}, "axon/verify/MsgSubmitAIChallengeResponse", nil)
	cdc.RegisterConcrete(&MsgRevealAIChallengeResponse{}, "axon/verify/MsgRevealAIChallengeResponse", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
		&MsgAddStake{},
		&MsgReduceStake{},
		&MsgClaimReducedStake{},
		&MsgUpdateVerify{},
		&MsgHeartbeat{},
		&MsgDeregister{},
		&MsgSubmitAIChallengeResponse{},
		&MsgRevealAIChallengeResponse{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &Msg_ServiceDesc)
}

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewProtoCodec(types.NewInterfaceRegistry())
)
