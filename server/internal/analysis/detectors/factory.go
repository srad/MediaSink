package detectors

import (
	"fmt"
	"sync"

	"github.com/srad/mediasink/server/internal/analysis/detectors/highlight"
	"github.com/srad/mediasink/server/internal/analysis/detectors/onnx"
	"github.com/srad/mediasink/server/internal/analysis/detectors/scene"
)

var (
	sceneDetector      SceneDetector
	highlightDetector  HighlightDetector
	embeddingExtractor EmbeddingExtractor
	mutex              = &sync.Mutex{}
)

// DetectorType specifies which detection algorithm to use.
type DetectorType string

const (
	DetectorTypeOnnxMobileNetV4Large DetectorType = "onnx_mobilenetv4_conv_large"
)

// DetectorConfig holds configuration for detector selection.
type DetectorConfig struct {
	SceneDetector     DetectorType
	HighlightDetector DetectorType
}

// DefaultDetectorConfig returns the default detector configuration.
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		SceneDetector:     DetectorTypeOnnxMobileNetV4Large,
		HighlightDetector: DetectorTypeOnnxMobileNetV4Large,
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
	case DetectorTypeOnnxMobileNetV4Large:
		sceneDetector, err = scene.NewOnnxSceneDetector(onnx.DefaultModelName)
	default:
		return nil, fmt.Errorf("unknown scene detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return sceneDetector, nil
}

// CreateEmbeddingExtractor creates an embedding extractor for frame-level feature extraction.
// The extractor is cached after creation to avoid expensive model reloading.
func CreateEmbeddingExtractor(detectorType DetectorType) (EmbeddingExtractor, error) {
	mutex.Lock()
	defer mutex.Unlock()

	if embeddingExtractor != nil {
		return embeddingExtractor, nil
	}

	var err error
	switch detectorType {
	case DetectorTypeOnnxMobileNetV4Large:
		var detector SceneDetector
		detector, err = scene.NewOnnxSceneDetector(onnx.DefaultModelName)
		if err == nil {
			var ok bool
			embeddingExtractor, ok = detector.(EmbeddingExtractor)
			if !ok {
				return nil, fmt.Errorf("scene detector %s does not expose embedding extraction", detectorType)
			}
		}
	default:
		return nil, fmt.Errorf("unknown embedding extractor type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return embeddingExtractor, nil
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
	case DetectorTypeOnnxMobileNetV4Large:
		highlightDetector, err = highlight.NewOnnxHighlightDetector(onnx.DefaultModelName)
	default:
		return nil, fmt.Errorf("unknown highlight detector type: %s", detectorType)
	}

	if err != nil {
		return nil, err
	}

	return highlightDetector, nil
}
