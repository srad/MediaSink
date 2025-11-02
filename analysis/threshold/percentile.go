package threshold

import (
	"fmt"
	"sort"
)

// percentileThresholdMethod uses percentile-based approach
// Uses the Nth percentile as the threshold
type percentileThresholdMethod struct {
	percentile float64 // 0.0-1.0, default 0.10 for 10th percentile
}

var _ ThresholdMethod = (*percentileThresholdMethod)(nil)

// NewPercentileThresholdMethod creates a new percentile threshold method
func NewPercentileThresholdMethod(p float64) ThresholdMethod {
	if p < 0 || p > 1 {
		p = 0.10 // default to 10th percentile
	}
	return &percentileThresholdMethod{percentile: p}
}

func (m *percentileThresholdMethod) Name() string {
	return "percentile"
}

func (m *percentileThresholdMethod) Description() string {
	return fmt.Sprintf("%.0f%% percentile method", m.percentile*100)
}

func (m *percentileThresholdMethod) Calculate(scores []float64) (float64, error) {
	if len(scores) == 0 {
		return 0, fmt.Errorf("no scores provided")
	}

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	return m.percentile_value(sorted, m.percentile), nil
}

func (m *percentileThresholdMethod) percentile_value(sortedScores []float64, p float64) float64 {
	if len(sortedScores) == 0 {
		return 0
	}
	if len(sortedScores) == 1 {
		return sortedScores[0]
	}

	// Linear interpolation method
	index := p * float64(len(sortedScores)-1)
	lowerIndex := int(index)
	upperIndex := lowerIndex + 1
	fraction := index - float64(lowerIndex)

	if upperIndex >= len(sortedScores) {
		return sortedScores[len(sortedScores)-1]
	}

	return sortedScores[lowerIndex]*(1-fraction) + sortedScores[upperIndex]*fraction
}
