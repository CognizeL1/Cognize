package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrModelNotFound        = errors.Register(ModuleName, 1400, "model not found")
	ErrModelAlreadyExists   = errors.Register(ModuleName, 1401, "model already registered")
	ErrModelNotVerified   = errors.Register(ModuleName, 1402, "model not verified")
	ErrInferenceFailed   = errors.Register(ModuleName, 1403, "inference failed")
	ErrDataNotFound     = errors.Register(ModuleName, 1405, "data not found")
	ErrDataNotOwner    = errors.Register(ModuleName, 1406, "not the data owner")
	ErrPriceNotMet     = errors.Register(ModuleName, 1407, "asking price not met")
	ErrPredictionNotFound = errors.Register(ModuleName, 1408, "prediction not found")
	ErrPredictionResolved = errors.Register(ModuleName, 1409, "prediction already resolved")
	ErrPredictionInsufficientStake = errors.Register(ModuleName, 1410, "insufficient stake for prediction")
)

type Prediction struct {
	PredictionID  string    `json:"prediction_id"`
	Question     string    `json:"question"`
	Creator     string    `json:"creator"`
	Outcomes    []string  `json:"outcomes"`
	StakeAmount string    `json:"stake_amount"`
	ResolutionBlock int64   `json:"resolution_block"`
	ResolvedBy   string   `json:"resolved_by"`
	Winner     string   `json:"winner"`
	Status     string   `json:"status"`
	TotalPool  string   `json:"total_pool"`
	BlockTime  int64    `json:"block_time"`
	ResolvedAt int64    `json:"resolved_at"`
}

type PredictionBet struct {
	BetID       string    `json:"bet_id"`
	PredictionID string    `json:"prediction_id"`
	Bettor     string    `json:"bettor"`
	Outcome    string    `json:"outcome"`
	Amount    string    `json:"amount"`
	Odds      string    `json:"odds"`
	BlockTime  int64     `json:"block_time"`
}

func KeyAIModel(modelID string) []byte {
	return []byte("aimodel/" + modelID)
}

func KeyAIModelByustate(state string) []byte {
	return []byte("aimodel/state/" + state)
}

func KeyInference(proofID string) []byte {
	return []byte("inference/" + proofID)
}

func KeyDataset(datasetID string) []byte {
	return []byte("dataset/" + datasetID)
}

func KeyDatasetByustate(state string) []byte {
	return []byte("dataset/state/" + state)
}

func KeyDataSale(saleID string) []byte {
	return []byte("datasale/" + saleID)
}

func KeyPrediction(predictionID string) []byte {
	return []byte("prediction/" + predictionID)
}

func KeyPredictionBet(betID string) []byte {
	return []byte("prediction/bet/" + betID)
}