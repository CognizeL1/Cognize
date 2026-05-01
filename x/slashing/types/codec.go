package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegister{}, "axon/slashing/MsgRegister", nil)
	cdc.RegisterConcrete(&MsgAddStake{}, "axon/slashing/MsgAddStake", nil)
	cdc.RegisterConcrete(&MsgReduceStake{}, "axon/slashing/MsgReduceStake", nil)
	cdc.RegisterConcrete(&MsgClaimReducedStake{}, "axon/slashing/MsgClaimReducedStake", nil)
	cdc.RegisterConcrete(&MsgUpdateuslashing{}, "axon/slashing/MsgUpdateuslashing", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "axon/slashing/MsgHeartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregister{}, "axon/slashing/MsgDeregister", nil)
	cdc.RegisterConcrete(&MsgSubmitAIChallengeResponse{}, "axon/slashing/MsgSubmitAIChallengeResponse", nil)
	cdc.RegisterConcrete(&MsgRevealAIChallengeResponse{}, "axon/slashing/MsgRevealAIChallengeResponse", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
		&MsgAddStake{},
		&MsgReduceStake{},
		&MsgClaimReducedStake{},
		&MsgUpdateuslashing{},
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
