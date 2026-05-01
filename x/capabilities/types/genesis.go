package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		ucapabilitiess: []ucapabilities{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, capabilities := range gs.ucapabilitiess {
		if capabilities.Address == "" {
			return fmt.Errorf("capabilities address cannot be empty")
		}
		if seen[capabilities.Address] {
			return fmt.Errorf("duplicate capabilities address: %s", capabilities.Address)
		}
		seen[capabilities.Address] = true
		if capabilities.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("capabilities %s reputation %d exceeds max %d", capabilities.Address, capabilities.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
