package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmath "cosmossdk.io/math"

	"github.com/cognize/axon/poaa-engine/dag"
	"github.com/cognize/axon/poaa-engine/memdag"
	"github.com/cognize/axon/poaa-engine/store"
)

type Engine struct {
	store      *store.DAGStore
	memDAG     *memdag.MemDAG
	validators *ValidatorSet
	vrfKey     *VRFSecretKey
	config     *Config

	mu              sync.RWMutex
	currentRound    uint64
	lastFinalized   [32]byte
	isRunning       bool
	height          uint64
	rewardPool      sdkmath.LegacyDec
	totalReputation int64
}

type Config struct {
	BlockTime          time.Duration
	MaxValidators      int
	MinStake           sdkmath.Int
	ConfirmationDepth  uint64
	VRFSeed            [32]byte
	EnableBLS          bool
	EnableKZG          bool
	FinalityThreshold  sdkmath.LegacyDec
	ReputationDecay    sdkmath.LegacyDec
}

func DefaultConfig() *Config {
	return &Config{
		BlockTime:          2 * time.Second,
		MaxValidators:      100,
		MinStake:           sdkmath.NewInt(1000000),
		ConfirmationDepth:  6,
		EnableBLS:          true,
		EnableKZG:          true,
		FinalityThreshold:  sdkmath.LegacyNewDec(67),
		ReputationDecay:    sdkmath.LegacyNewDec(99).Quo(sdkmath.LegacyNewDec(100)),
	}
}

func NewEngine(store *store.DAGStore, memDAG *memdag.MemDAG, vrfKey *VRFSecretKey, config *Config) *Engine {
	if config == nil {
		config = DefaultConfig()
	}

	return &Engine{
		store:      store,
		memDAG:     memDAG,
		validators: NewValidatorSet(),
		vrfKey:     vrfKey,
		config:     config,
		rewardPool: sdkmath.LegacyZeroDec(),
	}
}

func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.isRunning {
		return fmt.Errorf("engine already running")
	}

	e.isRunning = true

	go e.runLoop(ctx)

	return nil
}

func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.isRunning = false
	return nil
}

func (e *Engine) runLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.BlockTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !e.isRunning {
				return
			}
			e.processRound()
		}
	}
}

func (e *Engine) processRound() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.currentRound++

	seed := ComputeVRFSeed(e.lastFinalized, e.currentRound, time.Now().Unix())
	proposer, err := e.validators.SelectProposer(seed[:], e.currentRound)
	if err != nil {
		return
	}

	vertex := &dag.Vertex{
		Parents:   e.getParents(),
		Timestamp: time.Now().Unix(),
		Index:     e.height + 1,
		Sender:    proposer.Address.String(),
		TxValue:   sdkmath.ZeroInt(),
		Layer:     dag.LayerSoft,
		Depth:     e.memDAG.GetDepth() + 1,
	}

	vertex.Hash = vertex.ComputeHash()

	e.addVertex(vertex)
}

func (e *Engine) getParents() [2][32]byte {
	tips := e.memDAG.GetTips()
	parents := [2][32]byte{}

	if len(tips) > 0 {
		copy(parents[0][:], tips[0].Hash[:])
	}
	if len(tips) > 1 {
		copy(parents[1][:], tips[1].Hash[:])
	}

	return parents
}

func (e *Engine) addVertex(v *dag.Vertex) error {
	if err := v.Validate(); err != nil {
		return err
	}

	if err := e.memDAG.AddVertex(v); err != nil {
		return err
	}

	if err := e.store.PutVertex(v); err != nil {
		return err
	}

	e.height = v.Index
	return nil
}

func (e *Engine) SubmitVertex(v *dag.Vertex) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.addVertex(v)
}

func (e *Engine) ConfirmVertex(hash [32]byte, agent string, reputation int64, weight sdkmath.LegacyDec) error {
	record := dag.ConfirmRecord{
		Agent:      agent,
		Reputation: reputation,
		Weight:     weight,
		Timestamp:  time.Now().Unix(),
	}

	return e.memDAG.ConfirmVertex(hash, record)
}

func (e *Engine) GetVertex(hash [32]byte) (*dag.Vertex, error) {
	v, ok := e.memDAG.GetVertex(hash)
	if ok {
		return v, nil
	}

	return e.store.GetVertex(hash)
}

func (e *Engine) GetTips() []*dag.Vertex {
	return e.memDAG.GetTips()
}

func (e *Engine) GetHeight() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.height
}

func (e *Engine) AddValidator(addr sdk.AccAddress, power sdkmath.LegacyDec, vrfPubKey *VRFPublicKey, reputation int64) {
	e.validators.AddValidator(Validator{
		Address:    addr,
		Power:      power,
		VRFPubKey:  vrfPubKey,
		Reputation: reputation,
	})
}

func (e *Engine) RemoveValidator(addr sdk.AccAddress) {
	e.validators.RemoveValidator(addr)
}

func (e *Engine) GetValidators() []Validator {
	return e.validators.Validators
}

func (e *Engine) GetCurrentRound() uint64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.currentRound
}

func (e *Engine) FinalizeVertex(hash [32]byte) error {
	v, err := e.GetVertex(hash)
	if err != nil {
		return err
	}

	v.Layer = dag.LayerHard
	v.FinalityAt = time.Now().Unix()

	return e.store.PutVertex(v)
}

func (e *Engine) CheckFinality(hash [32]byte) (bool, error) {
	v, err := e.GetVertex(hash)
	if err != nil {
		return false, err
	}

	if v.Layer >= dag.LayerHard {
		return true, nil
	}

	weight := e.memDAG.CalculateWeight(hash)
	totalPower := e.validators.TotalPower

	if totalPower.IsZero() {
		return false, nil
	}

	percentage := weight.Quo(totalPower).Mul(sdkmath.LegacyNewDec(100))

	return percentage.GTE(e.config.FinalityThreshold), nil
}

func (e *Engine) SelectAggregator(round uint64) (*Validator, error) {
	seed := ComputeVRFSeed(e.lastFinalized, round, time.Now().Unix())
	return e.validators.SelectProposer(seed[:], round)
}

func (e *Engine) ProcessAggregatedSignature(vertexHash [32]byte, aggSig *AggregatedSignature) error {
	v, err := e.GetVertex(vertexHash)
	if err != nil {
		return err
	}

	msg := ComputeMessageHash(vertexHash, e.currentRound)
	if !VerifyAggregatedSignature(msg, aggSig) {
		return fmt.Errorf("invalid aggregated signature")
	}

	v.AggSig = aggSig.Signature
	v.Layer = dag.LayerFast

	return e.store.PutVertex(v)
}

func (e *Engine) DistributeRewards() {
	e.mu.Lock()
	defer e.mu.Unlock()

	validators := e.validators.Validators
	if len(validators) == 0 || e.rewardPool.IsZero() {
		return
	}

	rewardPerValidator := e.rewardPool.Quo(sdkmath.LegacyNewDec(int64(len(validators))))

	for i := range validators {
		repFactor := sdkmath.LegacyNewDec(validators[i].Reputation)
		totalFactor := repFactor.Add(validators[i].Power)
		_ = totalFactor
		_ = rewardPerValidator
	}

	e.rewardPool = sdkmath.LegacyZeroDec()
}

func (e *Engine) DecayReputations() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i := range e.validators.Validators {
		newRep := sdkmath.LegacyNewDec(e.validators.Validators[i].Reputation).Mul(e.config.ReputationDecay)
		e.validators.Validators[i].Reputation = newRep.TruncateInt64()
	}
}

func (e *Engine) GetConfirmationStatus(hash [32]byte) (dag.ConfirmationLayer, error) {
	v, err := e.GetVertex(hash)
	if err != nil {
		return 0, err
	}
	return v.Layer, nil
}

func (e *Engine) PruneOldVertices(depth uint64) int {
	return e.memDAG.PruneBelow(depth)
}

func (e *Engine) GetTotalWeight(hash [32]byte) sdkmath.LegacyDec {
	return e.memDAG.CalculateWeight(hash)
}

func (e *Engine) SetLastFinalized(hash [32]byte) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.lastFinalized = hash
}

func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.isRunning
}

func (e *Engine) SetHeight(height uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.height = height
}

func (e *Engine) GetCurrentProposer() (*Validator, error) {
	seed := ComputeVRFSeed(e.lastFinalized, e.currentRound, time.Now().Unix())
	return e.validators.SelectProposer(seed[:], e.currentRound)
}

func (e *Engine) ValidateChain(startHash [32]byte) error {
	visited := make(map[[32]byte]bool)
	queue := [][32]byte{startHash}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		v, err := e.GetVertex(current)
		if err != nil {
			return err
		}

		if err := v.Validate(); err != nil {
			return err
		}

		for _, p := range v.Parents {
			if p != [32]byte{} {
				queue = append(queue, p)
			}
		}
	}

	return nil
}

func (e *Engine) GetState() (uint64, uint64, int) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.height, e.currentRound, e.memDAG.Size()
}

func (e *Engine) AddToRewardPool(amount sdkmath.LegacyDec) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rewardPool = e.rewardPool.Add(amount)
}

func (e *Engine) GetRewardPool() sdkmath.LegacyDec {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.rewardPool
}

func (e *Engine) GetTotalReputation() int64 {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.totalReputation
}
