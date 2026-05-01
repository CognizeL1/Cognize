package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegister{}, "axon/state/MsgRegister", nil)
	cdc.RegisterConcrete(&MsgAddStake{}, "axon/state/MsgAddStake", nil)
	cdc.RegisterConcrete(&MsgReduceStake{}, "axon/state/MsgReduceStake", nil)
	cdc.RegisterConcrete(&MsgClaimReducedStake{}, "axon/state/MsgClaimReducedStake", nil)
	cdc.RegisterConcrete(&MsgUpdateustate{}, "axon/state/MsgUpdateustate", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "axon/state/MsgHeartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregister{}, "axon/state/MsgDeregister", nil)
	cdc.RegisterConcrete(&MsgSubmitAIChallengeResponse{}, "axon/state/MsgSubmitAIChallengeResponse", nil)
	cdc.RegisterConcrete(&MsgRevealAIChallengeResponse{}, "axon/state/MsgRevealAIChallengeResponse", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
		&MsgAddStake{},
		&MsgReduceStake{},
		&MsgClaimReducedStake{},
		&MsgUpdateustate{},
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
