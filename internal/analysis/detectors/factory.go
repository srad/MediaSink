package detectors

import (
	"fmt"
	"sync"

	"github.com/srad/mediasink/internal/analysis/detectors/highlight"
	"github.com/srad/mediasink/internal/analysis/detectors/scene"
)

var (
	sceneDetector     SceneDetector
	highlightDetector HighlightDetector
	mutex             = &sync.Mutex{}
)

// DetectorType specifies which detection algorithm to use.
type DetectorType string

const (
	DetectorTypeOnnxMobileNetV3Large DetectorType = "onnx_mobilenet_v3_large"
)

// DetectorConfig holds configuration for detector selection.
type DetectorConfig struct {
	SceneDetector     DetectorType
	HighlightDetector DetectorType
}

// DefaultDetectorConfig returns the default detector configuration.
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		SceneDetector:     DetectorTypeOnnxMobileNetV3Large,
		HighlightDetector: DetectorTypeOnnxMobileNetV3Large,
	}
}

// CreateSceneDetector creates a scene detector based on configuration.
// The detector is cached after creation to avoid expensive model reloading.
func CreateSceneDetector(detectorType DetectorType) (SceneDetector, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if sceneDetector != nil {
		return sceneDetector, nil
	}

	var err error
	switch detectorType {
	case DetectorTypeOnnxMobileNetV3Large:
		sceneDetector, err = scene.NewOnnxSceneDetector("mobilenet_v3_large")
	default:
		return nil, fmt.Errorf("unknown scene detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return sceneDetector, nil
}

// CreateHighlightDetector creates a highlight detector based on configuration.
// The detector is cached after creation to avoid expensive model reloading.
func CreateHighlightDetector(detectorType DetectorType) (HighlightDetector, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if highlightDetector != nil {
		return highlightDetector, nil
	}

	var err error
	switch detectorType {
	case DetectorTypeOnnxMobileNetV3Large:
		highlightDetector, err = highlight.NewOnnxHighlightDetector("mobilenet_v3_large")
	default:
		return nil, fmt.Errorf("unknown highlight detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return highlightDetector, nil
}

// CreateDetectors creates both scene and highlight detectors based on configuration.
func CreateDetectors(config *DetectorConfig) (SceneDetector, HighlightDetector, error) {
	sceneDetector, err := CreateSceneDetector(config.SceneDetector)
	if err != nil {
		return nil, nil, err
	}

	highlightDetector, err := CreateHighlightDetector(config.HighlightDetector)
	if err != nil {
		return nil, nil, err
	}

	return sceneDetector, highlightDetector, nil
}

// AvailableSceneDetectors returns the list of available scene detector names.
func AvailableSceneDetectors() []string {
	return []string{
		string(DetectorTypeOnnxMobileNetV3Large),
	}
}

// AvailableHighlightDetectors returns the list of available highlight detector names.
func AvailableHighlightDetectors() []string {
	return []string{
		string(DetectorTypeOnnxMobileNetV3Large),
	}
}
