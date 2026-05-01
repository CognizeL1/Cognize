package genesis

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	"github.com/cognize/axon/poaa-engine/consensus"
	"github.com/cognize/axon/poaa-engine/dag"
	"github.com/cognize/axon/poaa-engine/memdag"
	"github.com/cognize/axon/poaa-engine/store"
)

type GenesisConfig struct {
	ChainID        string             `json:"chain_id"`
	InitialHeight  uint64             `json:"initial_height"`
	GenesisTime    int64              `json:"genesis_time"`
	Validators     []GenesisValidator `json:"validators"`
	InitialSupply  string             `json:"initial_supply"`
	BlockTime      int64              `json:"block_time"`
	MaxValidators  int                `json:"max_validators"`
	MinStake       string             `json:"min_stake"`
}

type GenesisValidator struct {
	Address    string `json:"address"`
	PubKey      string `json:"pub_key"`
	Power      string `json:"power"`
	Reputation int64  `json:"reputation"`
}

type Genesis struct {
	Config    GenesisConfig    `json:"config"`
	Validator *consensus.ValidatorSet `json:"-"`
	Engine    *consensus.Engine       `json:"-"`
	Store     *store.DAGStore         `json:"-"`
	MemDAG    *memdag.MemDAG          `json:"-"`
}

func LoadGenesis(path string) (*Genesis, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read genesis file: %w", err)
	}

	var config GenesisConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse genesis: %w", err)
	}

	return &Genesis{Config: config}, nil
}

func (g *Genesis) Initialize(storePath string, vrfKey *consensus.VRFSecretKey) error {
	dagStore, err := store.NewDAGStore(storePath)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	g.Store = dagStore

	memDAG, err := memdag.NewMemDAG()
	if err != nil {
		return fmt.Errorf("failed to create mem DAG: %w", err)
	}
	g.MemDAG = memDAG

	validatorSet := consensus.NewValidatorSet()
	for _, gv := range g.Config.Validators {
		addr, err := sdk.AccAddressFromBech32(gv.Address)
		if err != nil {
			continue
		}

		power, _ := sdkmath.LegacyNewDecFromStr(gv.Power)
		pubKey := &consensus.VRFPublicKey{}

		validator := consensus.Validator{
			Address:    addr,
			Power:      power,
			VRFPubKey:  pubKey,
			Reputation: gv.Reputation,
		}
		validatorSet.AddValidator(validator)
	}
	g.Validator = validatorSet

	config := consensus.DefaultConfig()
	config.MaxValidators = g.Config.MaxValidators
	config.BlockTime = time.Duration(g.Config.BlockTime) * time.Second
	minStake, _ := sdkmath.LegacyNewDecFromStr(g.Config.MinStake)
	if minStake.IsNil() {
		minStake = sdkmath.LegacyZeroDec()
	}
	config.MinStake = minStake.TruncateInt()

	g.Engine = consensus.NewEngine(dagStore, memDAG, vrfKey, config)

	for _, v := range validatorSet.Validators {
		g.Engine.AddValidator(v.Address, v.Power, v.VRFPubKey, v.Reputation)
	}

	return g.createGenesisVertex()
}

func (g *Genesis) createGenesisVertex() error {
	genesisVertex := &dag.Vertex{
		Timestamp: g.Config.GenesisTime,
		Index:     g.Config.InitialHeight,
		Sender:    "genesis",
		TxValue:   sdkmath.ZeroInt(),
		Layer:     dag.LayerArchive,
		Confirmed: true,
		Depth:     0,
	}

	genesisVertex.Hash = genesisVertex.ComputeHash()

	if err := g.MemDAG.AddVertex(genesisVertex); err != nil {
		return fmt.Errorf("failed to add genesis vertex: %w", err)
	}

	if err := g.Store.PutVertex(genesisVertex); err != nil {
		return fmt.Errorf("failed to store genesis vertex: %w", err)
	}

	g.Engine.SetHeight(g.Config.InitialHeight)
	g.Engine.SetLastFinalized(genesisVertex.Hash)

	return nil
}

func CreateDefaultGenesis(chainID string) *GenesisConfig {
	return &GenesisConfig{
		ChainID:       chainID,
		InitialHeight: 1,
		GenesisTime:   time.Now().Unix(),
		Validators:    make([]GenesisValidator, 0),
		InitialSupply: "1000000000000000000",
		BlockTime:    2,
		MaxValidators: 100,
		MinStake:     "1000000",
	}
}

func (g *GenesisConfig) AddValidator(address, pubKey, power string, reputation int64) {
	g.Validators = append(g.Validators, GenesisValidator{
		Address:    address,
		PubKey:      pubKey,
		Power:      power,
		Reputation: reputation,
	})
}

func (g *GenesisConfig) Save(path string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write genesis file: %w", err)
	}

	return nil
}

func ValidateGenesis(config *GenesisConfig) error {
	if config.ChainID == "" {
		return fmt.Errorf("chain_id cannot be empty")
	}

	if config.InitialHeight == 0 {
		return fmt.Errorf("initial_height must be greater than 0")
	}

	if config.GenesisTime == 0 {
		return fmt.Errorf("genesis_time cannot be 0")
	}

	if len(config.Validators) == 0 {
		return fmt.Errorf("at least one validator required")
	}

	for i, v := range config.Validators {
		if v.Address == "" {
			return fmt.Errorf("validator %d: address cannot be empty", i)
		}

		_, err := sdk.AccAddressFromBech32(v.Address)
		if err != nil {
			return fmt.Errorf("validator %d: invalid address: %w", i, err)
		}

		if v.Power == "" {
			return fmt.Errorf("validator %d: power cannot be empty", i)
		}
	}

	return nil
}

func (g *Genesis) GetGenesisVertex() (*dag.Vertex, error) {
	return g.Store.GetVertex(g.Engine.GetTips()[0].Hash)
}

func BootstrapNode(chainID, genesisPath, storePath string, vrfKey *consensus.VRFSecretKey) (*consensus.Engine, error) {
	genesis, err := LoadGenesis(genesisPath)
	if err != nil {
		config := CreateDefaultGenesis(chainID)
		genesis = &Genesis{Config: *config}
	}

	if err := genesis.Initialize(storePath, vrfKey); err != nil {
		return nil, fmt.Errorf("failed to initialize genesis: %w", err)
	}

	return genesis.Engine, nil
}

func (g *Genesis) Close() error {
	if g.Store != nil {
		g.Store.Close()
	}
	return nil
}

func (g *Genesis) GetChainID() string {
	return g.Config.ChainID
}

func (g *Genesis) GetInitialHeight() uint64 {
	return g.Config.InitialHeight
}

func (g *Genesis) GetGenesisTime() int64 {
	return g.Config.GenesisTime
}

func (g *Genesis) GetValidators() []GenesisValidator {
	return g.Config.Validators
}
