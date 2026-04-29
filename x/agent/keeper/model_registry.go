package keeper

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cognize/axon/x/agent/types"
)

const (
	ModelVerificationFee = 10
	MaxPredictionDuration = 432000
)

type AIModelJSON struct {
	ModelID          string `json:"model_id"`
	AgentAddress    string `json:"agent_address"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	Architecture   string `json:"architecture"`
	Parameters     uint64 `json:"parameters"`
	InputSchema    string `json:"input_schema"`
	OutputSchema   string `json:"output_schema"`
	Hash           string `json:"hash"`
	MerkleRoot     string `json:"merkle_root"`
	Price          string `json:"price"`
	InferencePrice string `json:"inference_price"`
	Verified       bool  `json:"verified"`
	VerifiedAt    int64  `json:"verified_at"`
	VerifiedBy    string `json:"verified_by"`
	TotalInferences uint64 `json:"total_inferences"`
	TotalRevenue  string `json:"total_revenue"`
	LastInferenceAt int64  `json:"last_inference_at"`
	CreatedAt     int64  `json:"created_at"`
	Status       string `json:"status"`
}

type InferenceProofJSON struct {
	ProofID      string `json:"proof_id"`
	ModelID     string `json:"model_id"`
	InputHash   string `json:"input_hash"`
	OutputHash  string `json:"output_hash"`
	BlockHeight int64  `json:"block_height"`
	Timestamp  int64  `json:"timestamp"`
}

type DatasetJSON struct {
	DatasetID    string `json:"dataset_id"`
	AgentAddress string `json:"agent_address"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        uint64  `json:"size"`
	Format      string `json:"format"`
	Price       string `json:"price"`
	Samples     uint64 `json:"samples"`
	Anonymized  bool   `json:"anonymized"`
	Verified    bool   `json:"verified"`
	Downloads   uint64 `json:"downloads"`
	Revenue     string `json:"revenue"`
	CreatedAt   int64  `json:"created_at"`
	Status     string `json:"status"`
}

type PredictionJSON struct {
	PredictionID   string   `json:"prediction_id"`
	Question       string   `json:"question"`
	Creator        string   `json:"creator"`
	Outcomes       []string `json:"outcomes"`
	StakeAmount   string   `json:"stake_amount"`
	ResolutionBlock int64  `json:"resolution_block"`
	ResolvedBy    string   `json:"resolved_by"`
	Winner        string   `json:"winner"`
	Status        string   `json:"status"`
	TotalPool     string   `json:"total_pool"`
	BlockTime     int64   `json:"block_time"`
	ResolvedAt   int64    `json:"resolved_at"`
}

type PredictionBetJSON struct {
	BetID         string `json:"bet_id"`
	PredictionID string `json:"prediction_id"`
	Bettor        string `json:"bettor"`
	Outcome       string `json:"outcome"`
	Amount        string `json:"amount"`
	Odds          string `json:"odds"`
	BlockTime     int64  `json:"block_time"`
}

func (k Keeper) RegisterAIModel(ctx sdk.Context, agent, name, version, architecture, inputSchema, outputSchema, price, inferencePrice string, params uint64) (string, error) {
	store := ctx.KVStore(k.storeKey)

	modelID := generateModelID(agent, name, version)

	model := AIModelJSON{
		ModelID:         modelID,
		AgentAddress:    agent,
		Name:           name,
		Version:        version,
		Architecture:   architecture,
		Parameters:     params,
		InputSchema:    inputSchema,
		OutputSchema:   outputSchema,
		Price:          price,
		InferencePrice: inferencePrice,
		Verified:      false,
		TotalInferences: 0,
		TotalRevenue:  "0",
		CreatedAt:     ctx.BlockTime().Unix(),
		Status:        "active",
	}

	bz, _ := json.Marshal(&model)
	store.Set(types.KeyAIModel(modelID), bz)

	return modelID, nil
}

func (k Keeper) VerifyAIModel(ctx sdk.Context, modelID, verifier string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyAIModel(modelID))
	if bz == nil {
		return types.ErrModelNotFound
	}

	var model AIModelJSON
	json.Unmarshal(bz, &model)

	hash := sha256.Sum256([]byte(model.ModelID + model.Name + model.Version))
	model.Hash = hex.EncodeToString(hash[:])
	model.MerkleRoot = hex.EncodeToString(hash[:])
	model.Verified = true
	model.VerifiedAt = ctx.BlockTime().Unix()
	model.VerifiedBy = verifier

	bz, _ = json.Marshal(&model)
	store.Set(types.KeyAIModel(modelID), bz)

	k.recordRevenue(ctx, model.AgentAddress, math.NewInt(ModelVerificationFee))

	return nil
}

func (k Keeper) RecordInference(ctx sdk.Context, modelID, inputHash string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyAIModel(modelID))
	if bz == nil {
		return "", types.ErrModelNotFound
	}

	var model AIModelJSON
	json.Unmarshal(bz, &model)

	if !model.Verified {
		return "", types.ErrModelNotVerified
	}

	proofID := fmt.Sprintf("proof-%d-%s", ctx.BlockHeight(), modelID[:8])

	proof := InferenceProofJSON{
		ProofID:      proofID,
		ModelID:     modelID,
		InputHash:   inputHash,
		OutputHash:  model.MerkleRoot,
		BlockHeight: ctx.BlockHeight(),
		Timestamp:  ctx.BlockTime().Unix(),
	}

	bz, _ = json.Marshal(&proof)
	store.Set(types.KeyInference(proofID), bz)

	model.TotalInferences++
	model.LastInferenceAt = ctx.BlockTime().Unix()
	bz, _ = json.Marshal(&model)
	store.Set(types.KeyAIModel(modelID), bz)

	infPrice, _ := math.NewIntFromString(model.InferencePrice)
	k.recordRevenue(ctx, model.AgentAddress, infPrice)

	return proofID, nil
}

func (k Keeper) RegisterDataset(ctx sdk.Context, agent, name, description, format, price string, samples uint64, anonymized bool) (string, error) {
	datasetID := fmt.Sprintf("dataset-%d-%s", ctx.BlockHeight(), agent[:8])

	dataset := DatasetJSON{
		DatasetID:     datasetID,
		AgentAddress: agent,
		Name:         name,
		Description: description,
		Format:      format,
		Price:       price,
		Samples:     samples,
		Anonymized:  anonymized,
		Verified:   false,
		Downloads:  0,
		Revenue:   "0",
		CreatedAt: ctx.BlockTime().Unix(),
		Status:   "active",
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&dataset)
	store.Set(types.KeyDataset(datasetID), bz)

	return datasetID, nil
}

func (k Keeper) PurchaseDataset(ctx sdk.Context, datasetID, buyer, amount string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyDataset(datasetID))
	if bz == nil {
		return types.ErrDataNotFound
	}

	var dataset DatasetJSON
	json.Unmarshal(bz, &dataset)

	price, _ := math.NewIntFromString(dataset.Price)
	payAmt, _ := math.NewIntFromString(amount)

	if payAmt.LT(price) {
		return types.ErrPriceNotMet
	}

	dataset.Downloads++
	revenue, _ := math.NewIntFromString(dataset.Revenue)
	dataset.Revenue = revenue.Add(payAmt).String()

	bz, _ = json.Marshal(&dataset)
	store.Set(types.KeyDataset(datasetID), bz)

	sellerAddr := sdk.MustAccAddressFromBech32(dataset.AgentAddress)
	fee := payAmt.Mul(math.NewInt(25)).Quo(math.NewInt(10000))
	sellerPayout := payAmt.Sub(fee)

	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin("cognize", sellerPayout))); err != nil {
		return err
	}
	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sellerAddr, sdk.NewCoins(sdk.NewCoin("cognize", sellerPayout)))
}

func (k Keeper) CreatePrediction(ctx sdk.Context, creator, question string, outcomes []string, stakeAmount string, durationBlocks int64) (string, error) {
	stake, _ := math.NewIntFromString(stakeAmount)
	if stake.LT(math.NewInt(100)) {
		return "", types.ErrPredictionInsufficientStake
	}

	_, found := k.GetAgent(ctx, creator)
	if !found {
		return "", types.ErrAgentNotFound
	}

	predictionID := fmt.Sprintf("pred-%d-%s", ctx.BlockHeight(), creator[:8])

	prediction := PredictionJSON{
		PredictionID:    predictionID,
		Question:      question,
		Creator:       creator,
		Outcomes:      outcomes,
		StakeAmount:  stakeAmount,
		ResolutionBlock: ctx.BlockHeight() + durationBlocks,
		Status:       "open",
		TotalPool:    stakeAmount,
		BlockTime:   ctx.BlockTime().Unix(),
	}

	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(&prediction)
	store.Set(types.KeyPrediction(predictionID), bz)

	return predictionID, nil
}

func (k Keeper) PlacePredictionBet(ctx sdk.Context, predictionID, bettor, outcome, amount string) (string, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrediction(predictionID))
	if bz == nil {
		return "", types.ErrPredictionNotFound
	}

	var prediction PredictionJSON
	json.Unmarshal(bz, &prediction)

	if prediction.Status != "open" {
		return "", types.ErrPredictionResolved
	}

	if ctx.BlockHeight() > prediction.ResolutionBlock {
		return "", types.ErrPredictionResolved
	}

	betID := fmt.Sprintf("bet-%d-%s", ctx.BlockHeight(), bettor[:8])

	bet := PredictionBetJSON{
		BetID:         betID,
		PredictionID: predictionID,
		Bettor:        bettor,
		Outcome:      outcome,
		Amount:       amount,
		Odds:         "1.0",
		BlockTime:    ctx.BlockTime().Unix(),
	}

	bz, _ = json.Marshal(&bet)
	store.Set(types.KeyPredictionBet(betID), bz)

	amt, _ := math.NewIntFromString(amount)
	pool, _ := math.NewIntFromString(prediction.TotalPool)
	prediction.TotalPool = pool.Add(amt).String()

	bz, _ = json.Marshal(&prediction)
	store.Set(types.KeyPrediction(predictionID), bz)

	return betID, nil
}

func (k Keeper) ResolvePrediction(ctx sdk.Context, predictionID, resolver, winner string) error {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyPrediction(predictionID))
	if bz == nil {
		return types.ErrPredictionNotFound
	}

	var prediction PredictionJSON
	json.Unmarshal(bz, &prediction)

	if prediction.Status != "open" {
		return types.ErrPredictionResolved
	}

	prediction.Status = "resolved"
	prediction.ResolvedBy = resolver
	prediction.Winner = winner
	prediction.ResolvedAt = ctx.BlockTime().Unix()

	bz, _ = json.Marshal(&prediction)
	store.Set(types.KeyPrediction(predictionID), bz)

	return nil
}

func generateModelID(agent, name, version string) string {
	data := fmt.Sprintf("%s:%s:%s", agent, name, version)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16])
}