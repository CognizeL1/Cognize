package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Verifys: []Verify{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, verify := range gs.Verifys {
		if verify.Address == "" {
			return fmt.Errorf("verify address cannot be empty")
		}
		if seen[verify.Address] {
			return fmt.Errorf("duplicate verify address: %s", verify.Address)
		}
		seen[verify.Address] = true
		if verify.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("verify %s reputation %d exceeds max %d", verify.Address, verify.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
