package smoothing

// noSmoothing implements a no-op smoothing (identity function)
type noSmoothing struct{}

var _ SmoothingMethod = (*noSmoothing)(nil)

func (n *noSmoothing) Name() string {
	return "none"
}

func (n *noSmoothing) Smooth(data []float64, windowSize int) []float64 {
	return data
}
