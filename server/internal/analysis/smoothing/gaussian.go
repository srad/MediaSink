package smoothing

import "math"

// gaussianSmoothing implements Gaussian blur smoothing (preserves edges better)
type gaussianSmoothing struct{}

var _ SmoothingMethod = (*gaussianSmoothing)(nil)

func (g *gaussianSmoothing) Name() string {
	return "gaussian"
}

func (g *gaussianSmoothing) Smooth(data []float64, windowSize int) []float64 {
	if windowSize <= 1 || len(data) <= windowSize {
		return data
	}

	smoothed := make([]float64, len(data))
	half := windowSize / 2
	sigma := float64(half) / 2.0 // Standard deviation based on window size

	// Pre-calculate Gaussian kernel weights
	kernelSum := 0.0
	weights := make([]float64, windowSize)
	for i := -half; i <= half; i++ {
		weight := math.Exp(-float64(i*i) / (2 * sigma * sigma))
		weights[i+half] = weight
		kernelSum += weight
	}

	// Apply Gaussian filter
	for i := 0; i < len(data); i++ {
		weightedSum := 0.0
		totalWeight := 0.0

		for j := -half; j <= half; j++ {
			idx := i + j
			if idx >= 0 && idx < len(data) {
				weight := weights[j+half]
				weightedSum += data[idx] * weight
				totalWeight += weight
			}
		}

		if totalWeight > 0 {
			smoothed[i] = weightedSum / totalWeight
		} else {
			smoothed[i] = data[i]
		}
	}

	return smoothed
}
