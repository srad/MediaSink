package responses

import "github.com/srad/mediasink/internal/db"

type EnqueueAllResponse struct {
	Enqueued int `json:"enqueued"`
}

type SimilarVideoMatch struct {
	Recording     *db.Recording `json:"recording"`
	Similarity    float64             `json:"similarity"`
	BestTimestamp float64             `json:"bestTimestamp"`
}

type VisualSearchResponse struct {
	SimilarityThreshold float64             `json:"similarityThreshold"`
	Limit               int                 `json:"limit"`
	Results             []SimilarVideoMatch `json:"results"`
}

type SimilarVideoGroup struct {
	GroupID       int                   `json:"groupId"`
	MaxSimilarity float64               `json:"maxSimilarity"`
	Videos        []*db.Recording `json:"videos"`
}

type SimilarityGroupsResponse struct {
	SimilarityThreshold float64             `json:"similarityThreshold"`
	GroupCount          int                 `json:"groupCount"`
	Groups              []SimilarVideoGroup `json:"groups"`
	AnalyzedCount       int                 `json:"analyzedCount"` // recordings with stored frame vectors
}
