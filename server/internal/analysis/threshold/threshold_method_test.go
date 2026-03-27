package threshold

import (
	"fmt"
	"testing"
)

// TestThresholdMethods demonstrates each threshold method on sample data
func TestThresholdMethods(t *testing.T) {
	// Simulate similarity scores from a video analysis
	// Values range from 0.0 to 1.0, lower values indicate more change
	sampleScores := []float64{
		0.95, 0.96, 0.94, 0.97, 0.95, // very similar frames
		0.92, 0.91, 0.93, 0.94, 0.96, // still similar
		0.45, 0.50, 0.55, 0.52, 0.48, // big changes (scene cuts)
		0.88, 0.90, 0.92, 0.89, 0.91, // back to similar
		0.35, 0.40, 0.38, 0.42, 0.36, // another scene cut
		0.94, 0.96, 0.95, 0.97, 0.96, // similar again
	}

	t.Logf("Testing with %d similarity scores\n", len(sampleScores))

	var threshold float64
	var err error

	// Test Statistical Method with different k values
	t.Log("\n=== Statistical Method ===")
	for _, k := range []float64{0.5, 1.0, 1.5} {
		method := NewStatisticalThresholdMethod(k)
		threshold, err = method.Calculate(sampleScores)
		if err != nil {
			t.Errorf("Statistical method failed: %v", err)
			continue
		}
		count := countBelowThreshold(sampleScores, threshold)
		t.Logf("%s: threshold=%.4f, detections=%d (%.1f%%)",
			method.Description(),
			threshold,
			count,
			float64(count)/float64(len(sampleScores))*100,
		)
	}

	// Test Percentile Method with different percentiles
	t.Log("\n=== Percentile Method ===")
	for _, p := range []float64{0.05, 0.10, 0.15, 0.20} {
		method := NewPercentileThresholdMethod(p)
		threshold, err = method.Calculate(sampleScores)
		if err != nil {
			t.Errorf("Percentile method failed: %v", err)
			continue
		}
		count := countBelowThreshold(sampleScores, threshold)
		t.Logf("%s: threshold=%.4f, detections=%d (%.1f%%)",
			method.Description(),
			threshold,
			count,
			float64(count)/float64(len(sampleScores))*100,
		)
	}

	// Test Otsu's Method
	t.Log("\n=== Otsu's Method ===")
	otsuMethod := NewOtsusThresholdMethod()
	threshold, err = otsuMethod.Calculate(sampleScores)
	if err != nil {
		t.Errorf("Otsu's method failed: %v", err)
	} else {
		count := countBelowThreshold(sampleScores, threshold)
		t.Logf("%s: threshold=%.4f, detections=%d (%.1f%%)",
			otsuMethod.Description(),
			threshold,
			count,
			float64(count)/float64(len(sampleScores))*100,
		)
	}

	// Test Knee Method
	t.Log("\n=== Knee Detection Method ===")
	kneeMethod := NewKneeThresholdMethod()
	threshold, err = kneeMethod.Calculate(sampleScores)
	if err != nil {
		t.Errorf("Knee method failed: %v", err)
	} else {
		count := countBelowThreshold(sampleScores, threshold)
		t.Logf("%s: threshold=%.4f, detections=%d (%.1f%%)",
			kneeMethod.Description(),
			threshold,
			count,
			float64(count)/float64(len(sampleScores))*100,
		)
	}

	// Test Analyzer with all methods
	t.Log("\n=== Full Analysis with All Methods ===")
	allMethods := []ThresholdMethod{
		NewStatisticalThresholdMethod(1.0),
		NewPercentileThresholdMethod(0.10),
		NewOtsusThresholdMethod(),
		NewKneeThresholdMethod(),
	}

	analysis, err := AnalyzeAllScores(sampleScores, allMethods...)
	if err != nil {
		t.Errorf("Analysis failed: %v", err)
	} else {
		t.Log(analysis.String())
	}
}

// countBelowThreshold counts how many scores are below the threshold
func countBelowThreshold(scores []float64, threshold float64) int {
	count := 0
	for _, score := range scores {
		if score < threshold {
			count++
		}
	}
	return count
}

// Example: How to use ThresholdAnalyzer in your detector
func TestThresholdAnalyzerUsage(t *testing.T) {
	t.Log("\n=== Example: Using ThresholdAnalyzer ===\n")

	// Suppose we extracted similarity scores from frame-to-frame analysis
	similarityScores := []float64{
		0.91, 0.92, 0.90, 0.93, 0.91,
		0.45, 0.40, 0.42,
		0.94, 0.95, 0.93, 0.94,
		0.30, 0.32, 0.35,
		0.96, 0.95, 0.97,
	}

	// Example 1: Using Otsu's method
	t.Log("Example 1: Otsu's Method (optimal for bimodal)")
	analyzer := NewThresholdAnalyzer(NewOtsusThresholdMethod())
	threshold, err := analyzer.GetThreshold(similarityScores)
	if err != nil {
		t.Errorf("Failed: %v", err)
	} else {
		t.Logf("Recommended threshold: %.4f", threshold)
	}

	// Example 2: Switching to Statistical method
	t.Log("\nExample 2: Statistical Method (mean - 1.0*stddev)")
	analyzer.SetMethod(NewStatisticalThresholdMethod(1.0))
	threshold, err = analyzer.GetThreshold(similarityScores)
	if err != nil {
		t.Errorf("Failed: %v", err)
	} else {
		t.Logf("Recommended threshold: %.4f", threshold)
	}

	// Example 3: Switching to Percentile method
	t.Log("\nExample 3: Percentile Method (10th percentile)")
	analyzer.SetMethod(NewPercentileThresholdMethod(0.10))
	threshold, err = analyzer.GetThreshold(similarityScores)
	if err != nil {
		t.Errorf("Failed: %v", err)
	} else {
		t.Logf("Recommended threshold: %.4f", threshold)
	}

	// Example 4: Full analysis
	t.Log("\nExample 4: Full Analysis Report")
	methods := []ThresholdMethod{
		NewOtsusThresholdMethod(),
		NewStatisticalThresholdMethod(1.0),
		NewPercentileThresholdMethod(0.10),
		NewKneeThresholdMethod(),
	}
	analysis, _ := AnalyzeAllScores(similarityScores, methods...)
	fmt.Println(analysis.String())
}
