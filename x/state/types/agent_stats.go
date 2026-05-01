package types

type ustateStatistics struct {
	Address              string  `json:"address"`
	Reputation          uint64  `json:"reputation"`
	Status              string  `json:"status"`
	TotalChallenges     uint64  `json:"total_challenges"`
	SuccessfulChallenges uint64 `json:"successful_challenges"`
	TotalResponses      uint64  `json:"total_responses"`
	LastChallengeEpoch  uint64  `json:"last_challenge_epoch"`
	ConsecutiveSuccesses uint64 `json:"consecutive_successes"`
	ConsecutiveFailures  uint64 `json:"consecutive_failures"`
	SuccessRate         float64 `json:"success_rate"`
}

type ReputationHistoryEntry struct {
	Epoch       uint64 `json:"epoch"`
	BlockHeight int64  `json:"block_height"`
	OldRep      uint64 `json:"old_rep"`
	NewRep      uint64 `json:"new_rep"`
	Delta       int64  `json:"delta"`
	Reason      string `json:"reason"`
}

type ChallengeMetrics struct {
	Epoch            uint64  `json:"epoch"`
	TotalResponders uint64  `json:"total_responders"`
	PassCount       uint64  `json:"pass_count"`
	PassRate        float64 `json:"pass_rate"`
	AverageScore    int64   `json:"average_score"`
	MinScore        int64   `json:"min_score"`
	MaxScore        int64   `json:"max_score"`
}

type QueryustateStatsRequest struct {
	Address string `json:"address"`
}

type QueryustateStatsResponse struct {
	Stats ustateStatistics `json:"stats"`
}

type QueryReputationHistoryRequest struct {
	Address string `json:"address"`
	Limit   uint64 `json:"limit"`
}

type QueryReputationHistoryResponse struct {
	History []ReputationHistoryEntry `json:"history"`
}

type QueryTopustatesRequest struct {
	SortBy string `json:"sort_by"`
	Limit  int    `json:"limit"`
}

type QueryTopustatesResponse struct {
	ustates []ustateStatistics `json:"states"`
}

type QueryustatesByCapabilityRequest struct {
	Capabilities []string `json:"capabilities"`
	MatchAll     bool     `json:"match_all"`
}

type QueryustatesByCapabilityResponse struct {
	ustates []ustate `json:"states"`
}

type QueryChallengeMetricsRequest struct {
	Epoch uint64 `json:"epoch"`
}

type QueryChallengeMetricsResponse struct {
	Metrics ChallengeMetrics `json:"metrics"`
}

type QueryustateChallengeHistoryRequest struct {
	Address string `json:"address"`
	Limit   uint64 `json:"limit"`
}

type QueryustateChallengeHistoryResponse struct {
	Responses []AIResponse `json:"responses"`
}