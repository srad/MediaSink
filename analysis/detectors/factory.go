package detectors

import (
	"fmt"
	"sync"

	"github.com/srad/mediasink/analysis/detectors/highlight"
	"github.com/srad/mediasink/analysis/detectors/scene"
)

var (
	sceneDetector     SceneDetector
	highlightDetector HighlightDetector
	mutex             = &sync.Mutex{}
)

// DetectorType specifies which detection algorithm to use
type DetectorType string

const (
	// Scene Detectors
	DetectorTypeSSIM        DetectorType = "ssim"

	// Highlight Detectors
	DetectorTypeFrameDiff          DetectorType = "frame_diff"
	DetectorTypeTensorFlowMobileNetV2 DetectorType = "tensorflow_mobilenet_v2"
	DetectorTypeTensorFlowMobileNetV3Large DetectorType = "tensorflow_mobilenet_v3_large"
	DetectorTypeTensorFlowMobileViT      DetectorType = "tensorflow_mobilevit"
)

// DetectorConfig holds configuration for detector selection
type DetectorConfig struct {
	SceneDetector      DetectorType
	HighlightDetector  DetectorType
}

// DefaultDetectorConfig returns the default detector configuration
// Uses DeepLearning for scenes and highlights by default.
// To switch back to the old detectors, change the values below.
// For example, to use SSIM for scenes and FrameDiff for highlights, change to:
// SceneDetector:     DetectorTypeSSIM,
// HighlightDetector: DetectorTypeFrameDiff,
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		SceneDetector:     DetectorTypeTensorFlowMobileNetV3Large,
		HighlightDetector: DetectorTypeTensorFlowMobileNetV3Large,
	}
}

// CreateSceneDetector creates a scene detector based on configuration
// Caches the detector after creation to avoid expensive model reloading
func CreateSceneDetector(detectorType DetectorType) (SceneDetector, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if sceneDetector != nil {
		return sceneDetector, nil
	}

	var err error
	switch detectorType {
	case DetectorTypeSSIM:
		sceneDetector = scene.NewSSIMSceneDetector()
	case DetectorTypeTensorFlowMobileNetV2:
		sceneDetector, err = scene.NewTensorFlowSceneDetector("mobilenet_v2")
	case DetectorTypeTensorFlowMobileNetV3Large:
		sceneDetector, err = scene.NewTensorFlowSceneDetector("mobilenet_v3_large")
	case DetectorTypeTensorFlowMobileViT:
		sceneDetector, err = scene.NewTensorFlowSceneDetector("mobilevit")
	default:
		return nil, fmt.Errorf("unknown scene detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return sceneDetector, nil
}

// CreateHighlightDetector creates a highlight detector based on configuration
// Caches the detector after creation to avoid expensive model reloading
func CreateHighlightDetector(detectorType DetectorType) (HighlightDetector, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if highlightDetector != nil {
		return highlightDetector, nil
	}

	var err error
	switch detectorType {
	case DetectorTypeFrameDiff:
		highlightDetector = highlight.NewFrameDiffHighlightDetector()
	case DetectorTypeTensorFlowMobileNetV2:
		highlightDetector, err = highlight.NewTensorFlowHighlightDetector("mobilenet_v2")
	case DetectorTypeTensorFlowMobileNetV3Large:
		highlightDetector, err = highlight.NewTensorFlowHighlightDetector("mobilenet_v3_large")
	case DetectorTypeTensorFlowMobileViT:
		highlightDetector, err = highlight.NewTensorFlowHighlightDetector("mobilevit")
	default:
		return nil, fmt.Errorf("unknown highlight detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return highlightDetector, nil
}

// CreateDetectors creates both scene and highlight detectors based on configuration
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

// AvailableSceneDetectors returns list of available scene detector names
func AvailableSceneDetectors() []string {
	return []string{
		string(DetectorTypeSSIM),
		string(DetectorTypeTensorFlowMobileNetV2),
		string(DetectorTypeTensorFlowMobileNetV3Large),
		string(DetectorTypeTensorFlowMobileViT),
	}
}

// AvailableHighlightDetectors returns list of available highlight detector names
func AvailableHighlightDetectors() []string {
	return []string{
		string(DetectorTypeFrameDiff),
		string(DetectorTypeTensorFlowMobileNetV2),
		string(DetectorTypeTensorFlowMobileNetV3Large),
		string(DetectorTypeTensorFlowMobileViT),
	}
}
