package smoothing

// noSmoothing implements a no-op smoothing (identity function)
type noSmoothing struct{}

var _ SmoothingMethod = (*noSmoothing)(nil)

func (n *noSmoothing) Name() string {
	return "none"
}

// Smooth returns a copy of the input. Every SmoothingMethod returns a slice the
// caller owns, so the result is always safe to mutate.
func (n *noSmoothing) Smooth(data []float64, _ int) []float64 {
	return append([]float64(nil), data...)
}
