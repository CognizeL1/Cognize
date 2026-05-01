package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegister{}, "axon/channels/MsgRegister", nil)
	cdc.RegisterConcrete(&MsgAddStake{}, "axon/channels/MsgAddStake", nil)
	cdc.RegisterConcrete(&MsgReduceStake{}, "axon/channels/MsgReduceStake", nil)
	cdc.RegisterConcrete(&MsgClaimReducedStake{}, "axon/channels/MsgClaimReducedStake", nil)
	cdc.RegisterConcrete(&MsgUpdateChannels{}, "axon/channels/MsgUpdateChannels", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "axon/channels/MsgHeartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregister{}, "axon/channels/MsgDeregister", nil)
	cdc.RegisterConcrete(&MsgSubmitAIChallengeResponse{}, "axon/channels/MsgSubmitAIChallengeResponse", nil)
	cdc.RegisterConcrete(&MsgRevealAIChallengeResponse{}, "axon/channels/MsgRevealAIChallengeResponse", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
		&MsgAddStake{},
		&MsgReduceStake{},
		&MsgClaimReducedStake{},
		&MsgUpdateChannels{},
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
