package threshold

import (
	"fmt"
	"math"
)

// statisticalThresholdMethod uses mean - k*stddev approach
// Lower k means more detections, higher k means fewer detections
type statisticalThresholdMethod struct {
	k float64 // default 1.0 for balanced detection
}

var _ ThresholdMethod = (*statisticalThresholdMethod)(nil)

// NewStatisticalThresholdMethod creates a new statistical threshold method
func NewStatisticalThresholdMethod(k float64) ThresholdMethod {
	if k <= 0 {
		k = 1.0
	}
	return &statisticalThresholdMethod{k: k}
}

func (m *statisticalThresholdMethod) Name() string {
	return "statistical"
}

func (m *statisticalThresholdMethod) Description() string {
	return fmt.Sprintf("Mean - %.1f * StdDev method (k=%.1f)", m.k, m.k)
}

func (m *statisticalThresholdMethod) Calculate(scores []float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}

	// Calculate mean and standard deviation
	sum := 0.0
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(len(scores))

	sumSquaredDiff := 0.0
	for _, s := range scores {
		diff := s - mean
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(len(scores)))

	threshold := mean - (m.k * stdDev)

	// Clamp to valid range [0, 1]
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 1 {
		threshold = 1
	}

	return threshold, nil
}
