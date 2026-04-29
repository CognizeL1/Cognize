package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegister{}, "cognize/agent/MsgRegister", nil)
	cdc.RegisterConcrete(&MsgAddStake{}, "cognize/agent/MsgAddStake", nil)
	cdc.RegisterConcrete(&MsgReduceStake{}, "cognize/agent/MsgReduceStake", nil)
	cdc.RegisterConcrete(&MsgClaimReducedStake{}, "cognize/agent/MsgClaimReducedStake", nil)
	cdc.RegisterConcrete(&MsgUpdateAgent{}, "cognize/agent/MsgUpdateAgent", nil)
	cdc.RegisterConcrete(&MsgHeartbeat{}, "cognize/agent/MsgHeartbeat", nil)
	cdc.RegisterConcrete(&MsgDeregister{}, "cognize/agent/MsgDeregister", nil)
	cdc.RegisterConcrete(&MsgSubmitAIChallengeResponse{}, "cognize/agent/MsgSubmitAIChallengeResponse", nil)
	cdc.RegisterConcrete(&MsgRevealAIChallengeResponse{}, "cognize/agent/MsgRevealAIChallengeResponse", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
		&MsgAddStake{},
		&MsgReduceStake{},
		&MsgClaimReducedStake{},
		&MsgUpdateAgent{},
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
