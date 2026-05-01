package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		uslashings: []uslashing{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, slashing := range gs.uslashings {
		if slashing.Address == "" {
			return fmt.Errorf("slashing address cannot be empty")
		}
		if seen[slashing.Address] {
			return fmt.Errorf("duplicate slashing address: %s", slashing.Address)
		}
		seen[slashing.Address] = true
		if slashing.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("slashing %s reputation %d exceeds max %d", slashing.Address, slashing.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
