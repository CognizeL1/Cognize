package abci

import (
	"context"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cognize/axon/poaa-engine/consensus"
	"github.com/cognize/axon/poaa-engine/dag"
	"github.com/cognize/axon/poaa-engine/memdag"
)

type ABCIAdapter struct {
	engine    *consensus.Engine
	memDAG    *memdag.MemDAG
	validator *consensus.ValidatorSet
}

func NewABCIAdapter(engine *consensus.Engine, memDAG *memdag.MemDAG, validator *consensus.ValidatorSet) *ABCIAdapter {
	return &ABCIAdapter{
		engine:    engine,
		memDAG:    memDAG,
		validator: validator,
	}
}

func (a *ABCIAdapter) Info(ctx context.Context, req *abci.RequestInfo) (*abci.ResponseInfo, error) {
	height := a.engine.GetHeight()
	return &abci.ResponseInfo{
		Data:            "cognize-poaa",
		Version:         "1.0.0",
		AppVersion:      1,
		LastBlockHeight: int64(height),
		LastBlockAppHash: []byte{},
	}, nil
}

func (a *ABCIAdapter) InitChain(ctx context.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	validators := make([]abci.ValidatorUpdate, 0)

	for _, v := range a.validator.Validators {
		validators = append(validators, abci.Ed25519ValidatorUpdate(v.Address, v.Power.TruncateInt().Int64()))
	}

	return &abci.ResponseInitChain{
		Validators: validators,
	}, nil
}

func (a *ABCIAdapter) FinalizeBlock(ctx context.Context, req *abci.RequestFinalizeBlock) (*abci.ResponseFinalizeBlock, error) {
	events := make([]abci.Event, 0)

	for _, tx := range req.Txs {
		if len(tx) == 0 {
			continue
		}

		vertex := &dag.Vertex{
			TxBytes:   tx,
			Timestamp: 0,
			Layer:     dag.LayerSoft,
			Depth:     a.memDAG.GetDepth() + 1,
		}

		vertex.Hash = vertex.ComputeHash()

		if err := a.engine.SubmitVertex(vertex); err != nil {
			events = append(events, abci.Event{
				Type: "tx_failed",
				Attributes: []abci.EventAttribute{
					{
						Key:   "hash",
						Value: fmt.Sprintf("%x", vertex.Hash),
					},
					{
						Key:   "error",
						Value: err.Error(),
					},
				},
			})
			continue
		}

		events = append(events, abci.Event{
			Type: "tx_succeeded",
			Attributes: []abci.EventAttribute{
				{
					Key:   "hash",
					Value: fmt.Sprintf("%x", vertex.Hash),
				},
			},
		})
	}

	validators := make([]abci.ValidatorUpdate, 0)
	for _, v := range a.validator.Validators {
		validators = append(validators, abci.Ed25519ValidatorUpdate(v.Address, v.Power.TruncateInt().Int64()))
	}

	return &abci.ResponseFinalizeBlock{
		Events:          events,
		ValidatorUpdates: validators,
	}, nil
}

func (a *ABCIAdapter) Commit(ctx context.Context, req *abci.RequestCommit) (*abci.ResponseCommit, error) {
	return &abci.ResponseCommit{
		RetainHeight: 0,
	}, nil
}

func (a *ABCIAdapter) Query(ctx context.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
	if len(req.Data) < 32 {
		return &abci.ResponseQuery{
			Code: 1,
			Log:  "invalid query: hash too short",
		}, nil
	}

	var hash [32]byte
	copy(hash[:], req.Data[:32])

	v, err := a.engine.GetVertex(hash)
	if err != nil {
		return &abci.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("vertex not found: %v", err),
		}, nil
	}

	data, err := v.ToBytes()
	if err != nil {
		return &abci.ResponseQuery{
			Code: 1,
			Log:  fmt.Sprintf("serialization error: %v", err),
		}, nil
	}

	return &abci.ResponseQuery{
		Code: 0,
		Value: data,
	}, nil
}

func (a *ABCIAdapter) CheckTx(ctx context.Context, req *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
	if len(req.Tx) == 0 {
		return &abci.ResponseCheckTx{
			Code: 1,
			Log:  "empty transaction",
		}, nil
	}

	return &abci.ResponseCheckTx{
		Code:      0,
		GasWanted: 100000,
	}, nil
}

func (a *ABCIAdapter) PrepareProposal(ctx context.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	return &abci.ResponsePrepareProposal{}, nil
}

func (a *ABCIAdapter) ProcessProposal(ctx context.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	return &abci.ResponseProcessProposal{
		Status: abci.ResponseProcessProposal_ACCEPT,
	}, nil
}

func (a *ABCIAdapter) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	return &abci.ResponseExtendVote{
		VoteExtension: []byte{},
	}, nil
}

func (a *ABCIAdapter) VerifyVoteExtension(ctx context.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	return &abci.ResponseVerifyVoteExtension{
		Status: abci.ResponseVerifyVoteExtension_ACCEPT,
	}, nil
}

func (a *ABCIAdapter) ListSnapshots(ctx context.Context, req *abci.RequestListSnapshots) (*abci.ResponseListSnapshots, error) {
	return &abci.ResponseListSnapshots{
		Snapshots: make([]*abci.Snapshot, 0),
	}, nil
}

func (a *ABCIAdapter) OfferSnapshot(ctx context.Context, req *abci.RequestOfferSnapshot) (*abci.ResponseOfferSnapshot, error) {
	return &abci.ResponseOfferSnapshot{
		Result: abci.ResponseOfferSnapshot_ACCEPT,
	}, nil
}

func (a *ABCIAdapter) LoadSnapshotChunk(ctx context.Context, req *abci.RequestLoadSnapshotChunk) (*abci.ResponseLoadSnapshotChunk, error) {
	return &abci.ResponseLoadSnapshotChunk{
		Chunk: []byte{},
	}, nil
}

func (a *ABCIAdapter) ApplySnapshotChunk(ctx context.Context, req *abci.RequestApplySnapshotChunk) (*abci.ResponseApplySnapshotChunk, error) {
	return &abci.ResponseApplySnapshotChunk{
		Result: abci.ResponseApplySnapshotChunk_ACCEPT,
	}, nil
}

func (a *ABCIAdapter) GetValidatorInfo() ([]abci.ValidatorUpdate, error) {
	updates := make([]abci.ValidatorUpdate, 0)

	for _, v := range a.validator.Validators {
		updates = append(updates, abci.Ed25519ValidatorUpdate(v.Address, v.Power.TruncateInt().Int64()))
	}

	return updates, nil
}

func (a *ABCIAdapter) GetLatestHeight() uint64 {
	return a.engine.GetHeight()
}

func (a *ABCIAdapter) GetVertex(hash [32]byte) (*dag.Vertex, error) {
	return a.engine.GetVertex(hash)
}

func (a *ABCIAdapter) GetTips() []*dag.Vertex {
	return a.engine.GetTips()
}

func (a *ABCIAdapter) ValidateVertex(v *dag.Vertex) error {
	return v.Validate()
}

func (a *ABCIAdapter) GetBlockByHeight(height uint64) (*dag.Vertex, error) {
	allVertices := a.memDAG.GetAllVertices()
	for _, v := range allVertices {
		if v.Index == height {
			return v, nil
		}
	}
	return nil, fmt.Errorf("vertex not found at height %d", height)
}

func (a *ABCIAdapter) PrepareDAGContext(req *abci.RequestFinalizeBlock) *dag.Context {
	return &dag.Context{
		Height:    uint64(req.Height),
		Timestamp: 0,
		Proposer:  req.ProposerAddress,
	}
}

func (a *ABCIAdapter) ProcessTransaction(txBytes []byte) (*dag.Vertex, error) {
	vertex := &dag.Vertex{
		TxBytes: txBytes,
		Timestamp: 0,
		Layer:   dag.LayerSoft,
	}

	vertex.Hash = vertex.ComputeHash()

	if err := a.engine.SubmitVertex(vertex); err != nil {
		return nil, err
	}

	return vertex, nil
}
