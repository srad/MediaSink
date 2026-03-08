package smoothing

// SmoothingMethod defines the interface for different smoothing algorithms
type SmoothingMethod interface {
	// Smooth applies the smoothing filter to the input data
	Smooth(data []float64, windowSize int) []float64
	// Name returns the name of the smoothing method
	Name() string
}
