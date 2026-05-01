package types

import "fmt"

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Channelss: []Channels{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, channels := range gs.Channelss {
		if channels.Address == "" {
			return fmt.Errorf("channels address cannot be empty")
		}
		if seen[channels.Address] {
			return fmt.Errorf("duplicate channels address: %s", channels.Address)
		}
		seen[channels.Address] = true
		if channels.Reputation > gs.Params.MaxReputation {
			return fmt.Errorf("channels %s reputation %d exceeds max %d", channels.Address, channels.Reputation, gs.Params.MaxReputation)
		}
	}
	return nil
}
