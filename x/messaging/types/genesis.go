package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Messagings: []Messaging{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, messaging := range gs.Messagings {
		if messaging.Address == "" {
			return fmt.Errorf("messaging address cannot be empty")
		}
		if seen[messaging.Address] {
			return fmt.Errorf("duplicate messaging address: %s", messaging.Address)
		}
		seen[messaging.Address] = true
		if messaging.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("messaging %s reputation %d exceeds max %d", messaging.Address, messaging.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
