package highlight

import (
	"github.com/srad/mediasink/analysis/threshold"
	"image"
	"math"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/analysis/metrics"
)

// frameDiffHighlightDetector detects highlights using frame difference analysis
type frameDiffHighlightDetector struct {
	thresholdMethod threshold.ThresholdMethod // Use statistical method by default
}

var _ HighlightDetector = (*frameDiffHighlightDetector)(nil)

// NewFrameDiffHighlightDetector creates a new frame difference highlight detector
func NewFrameDiffHighlightDetector() HighlightDetector {
	return &frameDiffHighlightDetector{
		thresholdMethod: threshold.NewStatisticalThresholdMethod(3.0), // k=3.0 for highlight detection
	}
}

// Name returns the detector name
func (f *frameDiffHighlightDetector) Name() string {
	return "FrameDifference"
}

// Close releases any resources held by the detector
func (f *frameDiffHighlightDetector) Close() error {
	return nil
}

// DetectHighlights detects highlights using frame difference analysis with adaptive threshold
func (f *frameDiffHighlightDetector) DetectHighlights(frames []image.Image, timestamps []float64) ([]database.HighlightInfo, error) {
	if len(frames) < 2 {
		return nil, nil
	}

	// First pass: collect all frame differences
	var differences []float64
	for i := 1; i < len(frames); i++ {
		magnitude := metrics.CalculateFrameDifference(frames[i-1], frames[i])
		differences = append(differences, magnitude)
	}

	// Calculate adaptive threshold
	// For frame differences, we want to detect HIGH differences (opposite of SSIM)
	// So we need to convert to similarity metric: invert differences to get similarities
	var invertedDifferences []float64
	for _, diff := range differences {
		invertedDifferences = append(invertedDifferences, 1.0-diff)
	}

	threshold, err := f.thresholdMethod.Calculate(invertedDifferences)
	if err != nil {
		log.Warnf("[FrameDiff] Failed to calculate adaptive threshold: %v, using fallback", err)
		threshold = 0.5 // Fallback threshold
	}

	// Convert back to difference threshold
	diffThreshold := 1.0 - threshold

	// Second pass: detect highlights using calculated threshold
	var highlights []database.HighlightInfo
	highlightCount := 0

	for i := 0; i < len(differences); i++ {
		magnitude := differences[i]

		// Record highlight if motion exceeds threshold
		if magnitude >= diffThreshold {
			highlightCount++
			highlights = append(highlights, database.HighlightInfo{
				Timestamp: timestamps[i+1],
				Intensity: math.Min(magnitude, 1.0),
				Type:      "motion",
			})
		}
	}

	triggerRate := float64(highlightCount) / float64(len(differences)) * 100.0
	log.Infof("[FrameDiff] Detected %d highlights from %d frames (adaptive threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		len(highlights), len(frames), diffThreshold, f.thresholdMethod.Name(), highlightCount, len(differences), triggerRate)

	return highlights, nil
}

