package scene

import (
	"fmt"
	"image"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/analysis/threshold"
	"github.com/srad/mediasink/analysis/metrics"
	"gonum.org/v1/gonum/mat"
)

// ssimSceneDetector detects scenes using Structural Similarity Index
type ssimSceneDetector struct {
	thresholdMethod threshold.ThresholdMethod // Use statistical method by default
}

var _ SceneDetector = (*ssimSceneDetector)(nil)

// NewSSIMSceneDetector creates a new SSIM-based scene detector
func NewSSIMSceneDetector() SceneDetector {
	return &ssimSceneDetector{
		thresholdMethod: threshold.NewStatisticalThresholdMethod(2.0), // k=2.0 for more sensitive scene detection
	}
}

// Name returns the detector name
func (s *ssimSceneDetector) Name() string {
	return "SSIM"
}

// Close releases any resources held by the detector
func (s *ssimSceneDetector) Close() error {
	return nil
}

// ExtractFeatures is not applicable for SSIM detector
func (s *ssimSceneDetector) ExtractFeatures(frame image.Image) (*mat.VecDense, error) {
	return nil, fmt.Errorf("ExtractFeatures is not supported by SSIM detector")
}

// DetectScenes detects scenes using SSIM comparison with adaptive threshold
func (s *ssimSceneDetector) DetectScenes(frames []image.Image, timestamps []float64) ([]database.SceneInfo, error) {
	if len(frames) < 2 {
		return nil, nil
	}

	// First pass: collect all SSIM similarity scores
	var similarities []float64
	for i := 1; i < len(frames); i++ {
		ssim := metrics.CalculateSSIM(frames[i-1], frames[i])
		similarities = append(similarities, ssim)
	}

	// Calculate adaptive threshold
	threshold, err := s.thresholdMethod.Calculate(similarities)
	if err != nil {
		log.Warnf("[SSIM] Failed to calculate adaptive threshold: %v, using fallback", err)
		threshold = 0.5 // Fallback threshold
	}

	// Second pass: detect scenes using calculated threshold
	var scenes []database.SceneInfo
	sceneStart := 0.0
	sceneChangeCount := 0

	for i := 0; i < len(similarities); i++ {
		ssim := similarities[i]
		intensity := 1.0 - ssim // Higher intensity = more change

		// Scene boundary detected
		if ssim < threshold {
			sceneChangeCount++
			scenes = append(scenes, database.SceneInfo{
				StartTime:       sceneStart,
				EndTime:         timestamps[i+1],
				ChangeIntensity: intensity,
			})
			sceneStart = timestamps[i+1]
		}
	}

	// Add final scene
	if len(frames) > 0 {
		scenes = append(scenes, database.SceneInfo{
			StartTime:       sceneStart,
			EndTime:         timestamps[len(timestamps)-1],
			ChangeIntensity: 0.0,
		})
	}

	totalComparisons := len(similarities)
	triggerRate := float64(sceneChangeCount) / float64(totalComparisons) * 100.0
	log.Infof("[SSIM] Detected %d scenes from %d frames (adaptive threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		len(scenes), len(frames), threshold, s.thresholdMethod.Name(), sceneChangeCount, totalComparisons, triggerRate)

	return scenes, nil
}

