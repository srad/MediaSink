package threshold

// ThresholdMethod defines the interface for different threshold calculation strategies
type ThresholdMethod interface {
	// Calculate returns the recommended threshold for the given similarity scores
	// Scores below this threshold are considered detections (scene changes or highlights)
	Calculate(scores []float64) (float64, error)

	// Name returns the name of this threshold method
	Name() string

	// Description returns a human-readable description of how this method works
	Description() string
}
