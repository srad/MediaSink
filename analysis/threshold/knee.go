package threshold

import (
	"fmt"
	"math"
	"sort"
)

// kneeThresholdMethod finds the elbow/knee point in the distribution curve
// Uses the point of maximum distance from the line connecting min and max
type kneeThresholdMethod struct{}

var _ ThresholdMethod = (*kneeThresholdMethod)(nil)

// NewKneeThresholdMethod creates a new knee detection threshold method
func NewKneeThresholdMethod() ThresholdMethod {
	return &kneeThresholdMethod{}
}

func (m *kneeThresholdMethod) Name() string {
	return "knee"
}

func (m *kneeThresholdMethod) Description() string {
	return "Knee/Elbow detection - finds inflection point in distribution"
}

func (m *kneeThresholdMethod) Calculate(scores []float64) (float64, error) {
	if len(scores) < 3 {
		return 0, fmt.Errorf("need at least 3 scores for knee detection")
	}

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	minScore := sorted[0]
	maxScore := sorted[len(sorted)-1]

	if minScore == maxScore {
		// All scores are identical
		return minScore, nil
	}

	// Find the point farthest from the line connecting min and max
	maxDistance := 0.0
	kneeIndex := 0

	for i := 0; i < len(sorted); i++ {
		// Calculate perpendicular distance from point to line
		x1, y1 := float64(0), float64(minScore)
		x2, y2 := float64(len(sorted)-1), float64(maxScore)
		x0, y0 := float64(i), float64(sorted[i])

		// Distance from point (x0,y0) to line through (x1,y1) and (x2,y2)
		numerator := math.Abs((y2-y1)*x0 - (x2-x1)*y0 + x2*y1 - y2*x1)
		denominator := math.Sqrt(math.Pow(y2-y1, 2) + math.Pow(x2-x1, 2))

		if denominator == 0 {
			continue
		}

		distance := numerator / denominator

		if distance > maxDistance {
			maxDistance = distance
			kneeIndex = i
		}
	}

	return sorted[kneeIndex], nil
}
