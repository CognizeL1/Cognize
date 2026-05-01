package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		uinferences: []uinference{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, inference := range gs.uinferences {
		if inference.Address == "" {
			return fmt.Errorf("inference address cannot be empty")
		}
		if seen[inference.Address] {
			return fmt.Errorf("duplicate inference address: %s", inference.Address)
		}
		seen[inference.Address] = true
		if inference.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("inference %s reputation %d exceeds max %d", inference.Address, inference.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
