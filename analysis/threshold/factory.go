package threshold

import "fmt"

// NewThresholdMethod creates a threshold method by name with optional parameters
func NewThresholdMethod(methodName string, args ...float64) (ThresholdMethod, error) {
	switch methodName {
	case "statistical":
		k := 1.0
		if len(args) > 0 {
			k = args[0]
		}
		return NewStatisticalThresholdMethod(k), nil
	case "percentile":
		p := 0.10
		if len(args) > 0 {
			p = args[0]
		}
		return NewPercentileThresholdMethod(p), nil
	case "otsu":
		return NewOtsusThresholdMethod(), nil
	case "knee":
		return NewKneeThresholdMethod(), nil
	default:
		return nil, fmt.Errorf("unknown threshold method: %s", methodName)
	}
}
