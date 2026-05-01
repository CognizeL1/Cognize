package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Oracles: []Oracle{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, oracle := range gs.Oracles {
		if oracle.Address == "" {
			return fmt.Errorf("oracle address cannot be empty")
		}
		if seen[oracle.Address] {
			return fmt.Errorf("duplicate oracle address: %s", oracle.Address)
		}
		seen[oracle.Address] = true
		if oracle.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("oracle %s reputation %d exceeds max %d", oracle.Address, oracle.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
