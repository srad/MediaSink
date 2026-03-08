package threshold

import (
	"fmt"
	"math"
	"sort"
)

// otsusThresholdMethod finds optimal threshold using Otsu's method
// Best when data has bimodal distribution (similar vs dissimilar)
type otsusThresholdMethod struct{}

var _ ThresholdMethod = (*otsusThresholdMethod)(nil)

// NewOtsusThresholdMethod creates a new Otsu's threshold method
func NewOtsusThresholdMethod() ThresholdMethod {
	return &otsusThresholdMethod{}
}

func (m *otsusThresholdMethod) Name() string {
	return "otsu"
}

func (m *otsusThresholdMethod) Description() string {
	return "Otsu's method - optimal threshold for bimodal distribution"
}

func (m *otsusThresholdMethod) Calculate(scores []float64) (float64, error) {
	if len(scores) < 2 {
		return 0, fmt.Errorf("need at least 2 scores for Otsu's method")
	}

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	maxVariance := 0.0
	optimalThreshold := sorted[0]

	// Try each unique score as a potential threshold
	for i := 0; i < len(sorted)-1; i++ {
		threshold := sorted[i]

		// Split scores into two groups: below and above threshold
		var below []float64
		var above []float64

		for _, score := range sorted {
			if score < threshold {
				below = append(below, score)
			} else {
				above = append(above, score)
			}
		}

		if len(below) == 0 || len(above) == 0 {
			continue
		}

		// Calculate means
		belowMean := 0.0
		for _, s := range below {
			belowMean += s
		}
		belowMean /= float64(len(below))

		aboveMean := 0.0
		for _, s := range above {
			aboveMean += s
		}
		aboveMean /= float64(len(above))

		// Calculate between-class variance
		w0 := float64(len(below)) / float64(len(sorted))
		w1 := float64(len(above)) / float64(len(sorted))
		variance := w0 * w1 * math.Pow(belowMean-aboveMean, 2)

		if variance > maxVariance {
			maxVariance = variance
			optimalThreshold = threshold
		}
	}

	return optimalThreshold, nil
}
