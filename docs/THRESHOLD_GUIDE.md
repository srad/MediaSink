# Adaptive Threshold Detection Guide

## Overview

Instead of using hardcoded thresholds for scene and highlight detection, you can now automatically determine optimal thresholds for each video using four different methods.

## Available Threshold Methods

### 1. **Statistical Method** - `NewStatisticalThresholdMethod(k)`
- **Formula**: `threshold = mean(scores) - k * std_dev(scores)`
- **How it works**: Uses statistical properties to find outliers
- **Parameters**:
  - `k=0.5`: More detections (sensitive)
  - `k=1.0`: Balanced (recommended default)
  - `k=1.5`: Fewer detections (conservative)
- **Best for**: Videos with consistent lighting and camera work
- **Pros**: Simple, fast, intuitive
- **Cons**: Assumes normal distribution

**Example output**: With k=1.0
```
threshold=0.5335 → 30% detections
```

---

### 2. **Percentile Method** - `NewPercentileThresholdMethod(p)`
- **How it works**: Uses the Nth percentile as the threshold
- **Parameters**:
  - `p=0.05`: Very conservative (5% detections)
  - `p=0.10`: Standard (10% detections) - recommended
  - `p=0.20`: Sensitive (20% detections)
- **Best for**: When you want a predictable number of detections
- **Pros**: Intuitive and predictable
- **Cons**: Ignores distribution shape

**Example output**: With p=0.10
```
threshold=0.3980 → 10% detections
```

---

### 3. **Otsu's Method** - `NewOtsusThresholdMethod()`
- **How it works**: Finds the optimal threshold that best separates two classes (similar vs dissimilar)
- **Assumptions**: Data has bimodal distribution (peaks at both low and high similarity)
- **Best for**: Most video content (scene cuts vs static content)
- **Pros**: Theoretically optimal for bimodal data, no parameters to tune
- **Cons**: Can fail with uniform or highly skewed distributions

**Example output**:
```
threshold=0.8800 → 33.3% detections
```

---

### 4. **Knee Detection** - `NewKneeThresholdMethod()`
- **How it works**: Finds the "inflection point" (elbow) in the distribution curve
- **Best for**: Discovering natural separation points in unknown data
- **Pros**: Adaptive, works with any distribution
- **Cons**: Slightly more complex computation

**Example output**:
```
threshold=0.8800 → 33.3% detections
```

---

## Comparison on Sample Data

| Method | Threshold | Detections | Use Case |
|--------|-----------|-----------|----------|
| Statistical (k=1.0) | 0.5335 | 30.0% | Balanced, general use |
| Percentile (p=0.10) | 0.3980 | 10.0% | Conservative, high precision |
| Otsu | 0.8800 | 33.3% | Bimodal data (most videos) |
| Knee | 0.8800 | 33.3% | Unknown distribution |

---

## Usage Examples

### Example 1: Using a Single Method

```go
package main

import "github.com/srad/mediasink/services/detectors"

// Extract similarity scores from your frame analysis
scores := []float64{0.95, 0.92, 0.45, 0.88, 0.35, 0.94, ...}

// Use Otsu's method (recommended for most videos)
method := detectors.NewOtsusThresholdMethod()
threshold, err := method.Calculate(scores)
if err != nil {
    log.Fatal(err)
}

// Now use this threshold for detection
for i, score := range scores {
    if score < threshold {
        fmt.Printf("Frame %d: Change detected (score=%.4f)\n", i, score)
    }
}
```

### Example 2: Switching Between Methods

```go
// Create analyzer with initial method
analyzer := detectors.NewThresholdAnalyzer(
    detectors.NewOtsusThresholdMethod(),
)

threshold1, _ := analyzer.GetThreshold(scores)
fmt.Printf("Otsu: %.4f\n", threshold1)

// Switch to statistical method
analyzer.SetMethod(detectors.NewStatisticalThresholdMethod(1.0))
threshold2, _ := analyzer.GetThreshold(scores)
fmt.Printf("Statistical: %.4f\n", threshold2)

// Switch to percentile
analyzer.SetMethod(detectors.NewPercentileThresholdMethod(0.10))
threshold3, _ := analyzer.GetThreshold(scores)
fmt.Printf("Percentile: %.4f\n", threshold3)
```

### Example 3: Analyze All Methods and Compare

```go
// Analyze with all methods
methods := []detectors.ThresholdMethod{
    detectors.NewOtsusThresholdMethod(),
    detectors.NewStatisticalThresholdMethod(1.0),
    detectors.NewPercentileThresholdMethod(0.10),
    detectors.NewKneeThresholdMethod(),
}

analysis, _ := detectors.AnalyzeAllScores(scores, methods...)
fmt.Println(analysis.String())

// Output shows:
// - Statistics (mean, std_dev, percentiles)
// - All threshold methods with detection counts
// - Comparison of results
```

---

## Integration with Detectors

To use adaptive thresholds in your detectors:

### Before (Hardcoded Threshold)
```go
const sceneThreshold = 0.75

func (d *SSIMSceneDetector) DetectScenes(frames []image.Image, timestamps []float64) ([]database.SceneInfo, error) {
    var similarities []float64
    for i := 1; i < len(frames); i++ {
        sim := SSIM(frames[i-1], frames[i])
        similarities = append(similarities, sim)

        if sim < sceneThreshold {  // ← Fixed threshold
            // Record scene change
        }
    }
}
```

### After (Adaptive Threshold)
```go
func (d *SSIMSceneDetector) DetectScenes(frames []image.Image, timestamps []float64) ([]database.SceneInfo, error) {
    var similarities []float64

    // First pass: collect all similarities
    for i := 1; i < len(frames); i++ {
        sim := SSIM(frames[i-1], frames[i])
        similarities = append(similarities, sim)
    }

    // Determine optimal threshold
    method := detectors.NewOtsusThresholdMethod()
    threshold, _ := method.Calculate(similarities)  // ← Adaptive threshold

    // Second pass: detect with optimal threshold
    for i, sim := range similarities {
        if sim < threshold {
            // Record scene change
        }
    }
}
```

---

## Recommendations

### For Scene Detection
- **Recommended**: Otsu's Method
- **Why**: Videos naturally have two modes (static content vs scene changes)
- **Fallback**: Statistical with k=1.0

### For Highlight Detection
- **Recommended**: Statistical with k=0.5-1.0
- **Why**: Highlights are typically 10-30% of video
- **Fallback**: Percentile with p=0.15-0.20

### For Unknown Scenarios
1. Try Otsu's method first (theoretically optimal)
2. Compare with Statistical (k=1.0)
3. Use Percentile (p=0.10) as conservative baseline
4. Knee method as adaptive fallback

---

## Performance Characteristics

| Method | Speed | Accuracy | Robustness |
|--------|-------|----------|-----------|
| Statistical | Very Fast | Good | Medium |
| Percentile | Very Fast | Good | High |
| Otsu | Fast | Excellent | Medium |
| Knee | Moderate | Good | High |

---

## Testing

Run the included tests to see all methods in action:

```bash
# Test all methods on sample data
go test ./services/detectors -run TestThresholdMethods -v

# Test usage examples
go test ./services/detectors -run TestThresholdAnalyzerUsage -v
```

---

## Advanced: Custom Threshold Methods

You can implement your own threshold method by implementing the `ThresholdMethod` interface:

```go
type MyCustomThresholdMethod struct {
    // your fields
}

func (m *MyCustomThresholdMethod) Name() string {
    return "custom"
}

func (m *MyCustomThresholdMethod) Description() string {
    return "My custom threshold method"
}

func (m *MyCustomThresholdMethod) Calculate(scores []float64) (float64, error) {
    // Your threshold calculation logic
    return threshold, nil
}

// Now use it
analyzer := detectors.NewThresholdAnalyzer(m)
threshold, _ := analyzer.GetThreshold(scores)
```
