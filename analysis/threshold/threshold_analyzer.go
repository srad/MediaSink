package threshold

import (
	"fmt"
	"math"
	"sort"
)

// ThresholdAnalyzer analyzes similarity scores using a specified threshold method
type ThresholdAnalyzer struct {
	method ThresholdMethod
}

// NewThresholdAnalyzer creates a new threshold analyzer with the specified method
func NewThresholdAnalyzer(method ThresholdMethod) *ThresholdAnalyzer {
	return &ThresholdAnalyzer{method: method}
}

// SetMethod changes the threshold calculation method
func (a *ThresholdAnalyzer) SetMethod(method ThresholdMethod) {
	a.method = method
}

// GetThreshold calculates the threshold using the current method
func (a *ThresholdAnalyzer) GetThreshold(scores []float64) (float64, error) {
	if a.method == nil {
		return 0, fmt.Errorf("no threshold method set")
	}
	return a.method.Calculate(scores)
}

// AnalyzeScores provides detailed analysis of similarity scores
// Returns statistics and optionally calculates thresholds using multiple methods
type ScoreAnalysis struct {
	Count  int
	Min    float64
	Max    float64
	Mean   float64
	StdDev float64
	Median float64
	P25    float64
	P75    float64

	// Threshold results
	Thresholds map[string]float64 // method name -> threshold value
	Detections map[string]int      // method name -> detection count
}

// AnalyzeAllScores returns detailed statistics about the scores
// and optionally calculates thresholds using all provided methods
func AnalyzeAllScores(scores []float64, methods ...ThresholdMethod) (*ScoreAnalysis, error) {
	if len(scores) == 0 {
		return nil, fmt.Errorf("no scores provided")
	}

	analysis := &ScoreAnalysis{
		Count:      len(scores),
		Thresholds: make(map[string]float64),
		Detections: make(map[string]int),
	}

	// Sort scores for percentile calculations
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	// Calculate statistics
	analysis.Min = sorted[0]
	analysis.Max = sorted[len(sorted)-1]

	sum := 0.0
	for _, s := range sorted {
		sum += s
	}
	analysis.Mean = sum / float64(len(sorted))

	// Standard deviation
	sumSquaredDiff := 0.0
	for _, s := range sorted {
		diff := s - analysis.Mean
		sumSquaredDiff += diff * diff
	}
	analysis.StdDev = math.Sqrt(sumSquaredDiff / float64(len(sorted)))

	// Median
	if len(sorted)%2 == 0 {
		analysis.Median = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	} else {
		analysis.Median = sorted[len(sorted)/2]
	}

	// Percentiles
	analysis.P25 = percentileValue(sorted, 0.25)
	analysis.P75 = percentileValue(sorted, 0.75)

	// Calculate thresholds for each method
	for _, method := range methods {
		threshold, err := method.Calculate(scores)
		if err != nil {
			continue
		}

		analysis.Thresholds[method.Name()] = threshold

		// Count detections (scores below threshold)
		count := 0
		for _, score := range scores {
			if score < threshold {
				count++
			}
		}
		analysis.Detections[method.Name()] = count
	}

	return analysis, nil
}

// percentileValue calculates the Nth percentile value
func percentileValue(sortedScores []float64, p float64) float64 {
	if len(sortedScores) == 0 {
		return 0
	}
	if len(sortedScores) == 1 {
		return sortedScores[0]
	}

	index := p * float64(len(sortedScores)-1)
	lowerIndex := int(index)
	upperIndex := lowerIndex + 1
	fraction := index - float64(lowerIndex)

	if upperIndex >= len(sortedScores) {
		return sortedScores[len(sortedScores)-1]
	}

	return sortedScores[lowerIndex]*(1-fraction) + sortedScores[upperIndex]*fraction
}

// String returns a formatted report of the score analysis
func (analysis *ScoreAnalysis) String() string {
	report := fmt.Sprintf(`
Score Analysis Report
=====================
Sample Size: %d pairs/frames
Score Range: [%.4f, %.4f]

Distribution Statistics:
  Mean:      %.4f
  Std Dev:   %.4f
  Median:    %.4f
  25th %%ile:  %.4f
  75th %%ile:  %.4f

`,
		analysis.Count,
		analysis.Min, analysis.Max,
		analysis.Mean,
		analysis.StdDev,
		analysis.Median,
		analysis.P25,
		analysis.P75,
	)

	if len(analysis.Thresholds) > 0 {
		report += "Threshold Methods:\n"
		for method, threshold := range analysis.Thresholds {
			detections := analysis.Detections[method]
			detectionRate := float64(detections) / float64(analysis.Count) * 100
			report += fmt.Sprintf("  %-12s: %.4f → %4d detections (%.1f%%)\n",
				method, threshold, detections, detectionRate)
		}
	}

	return report
}
