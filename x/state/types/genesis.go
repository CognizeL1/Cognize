package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		States: []State{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, state := range gs.States {
		if state.Address == "" {
			return fmt.Errorf("state address cannot be empty")
		}
		if seen[state.Address] {
			return fmt.Errorf("duplicate state address: %s", state.Address)
		}
		seen[state.Address] = true
		if state.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("state %s reputation %d exceeds max %d", state.Address, state.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
