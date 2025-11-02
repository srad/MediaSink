package smoothing

import "fmt"

// NewSmoothingMethod creates a smoothing method by name
func NewSmoothingMethod(methodName string) (SmoothingMethod, error) {
	switch methodName {
	case "moving-average":
		return &movingAverageSmoothing{}, nil
	case "gaussian":
		return &gaussianSmoothing{}, nil
	case "median":
		return &medianSmoothing{}, nil
	case "none":
		return &noSmoothing{}, nil
	default:
		return nil, fmt.Errorf("unknown smoothing method: %s", methodName)
	}
}

// DefaultSmoothingMethod returns the default smoothing method (median for best edge preservation)
func DefaultSmoothingMethod() SmoothingMethod {
	return &medianSmoothing{}
}
