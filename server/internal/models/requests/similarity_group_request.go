package requests

// SimilarityGroupRequest groups visually similar recordings.
type SimilarityGroupRequest struct {
	// Similarity threshold. Supports 0..1 and 0..100 (percent) formats.
	Similarity *float64 `json:"similarity"`

	// Optional subset. When empty, all recordings with frame vectors are considered.
	RecordingIDs []uint `json:"recordingIds"`

	// Hard cap on pairwise comparisons/edges considered.
	PairLimit int `json:"pairLimit"`

	// Include singleton groups (recordings without neighbors above threshold).
	IncludeSingletons bool `json:"includeSingletons"`
}
