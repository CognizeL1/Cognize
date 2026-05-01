package app

import (
	"github.com/cognize/axon/poaa-engine/abci"
	"github.com/cognize/axon/poaa-engine/consensus"
	"github.com/cognize/axon/poaa-engine/dag"
	"github.com/cognize/axon/poaa-engine/store"
	"github.com/cognize/axon/poaa-engine/memdag"
)

type PoAAIntegration struct {
	engine    *consensus.Engine
	adapter   *abci.ABCIAdapter
	memDAG    *memdag.MemDAG
	dagStore  *store.DAGStore
}

func NewPoAAIntegration(dataDir string) (*PoAAIntegration, error) {
	dagStore, err := store.NewDAGStore(dataDir + "/poaa/dag")
	if err != nil {
		return nil, err
	}

	memDAG, err := memdag.NewMemDAG()
	if err != nil {
		return nil, err
	}

	engine := consensus.NewEngine(dagStore, memDAG, nil, consensus.DefaultConfig())

	validatorSet := consensus.NewValidatorSet()
	adapter := abci.NewABCIAdapter(engine, memDAG, validatorSet)

	return &PoAAIntegration{
		engine:   engine,
		adapter:  adapter,
		memDAG:  memDAG,
		dagStore: dagStore,
	}, nil
}

func (p *PoAAIntegration) Start() error {
	return p.engine.Start(nil)
}

func (p *PoAAIntegration) Stop() error {
	return p.engine.Stop()
}

func (p *PoAAIntegration) GetABCIAdapter() *abci.ABCIAdapter {
	return p.adapter
}

func (p *PoAAIntegration) SubmitTransaction(txBytes []byte) (*dag.Vertex, error) {
	vertex := &dag.Vertex{
		TxBytes: txBytes,
		Layer:   dag.LayerSoft,
	}
	vertex.Hash = vertex.ComputeHash()
	return vertex, p.engine.SubmitVertex(vertex)
}
