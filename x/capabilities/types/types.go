package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BankKeeper interface {
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
}

type StakingKeeper interface {
	GetBondedValidatorsByPower(ctx sdk.Context) []interface{}
}

type Params struct {
	MaxCapabilitiesPerAgent uint64
}

type GenesisState struct {
	Params Params
}

func DefaultGenesisState() GenesisState {
	return GenesisState{
		Params: DefaultParams(),
	}
}

func DefaultParams() Params {
	return Params{
		MaxCapabilitiesPerAgent: 10,
	}
}

func (p Params) Validate() error {
	return nil
}

func ModuleName() string {
	return "capabilities"
}
