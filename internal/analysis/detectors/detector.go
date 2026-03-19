package detectors

import (
	"image"

	"github.com/srad/mediasink/internal/analysis/detectors/highlight"
	"github.com/srad/mediasink/internal/analysis/detectors/scene"
)

// SceneDetector is re-exported from scene package for backward compatibility
type SceneDetector = scene.SceneDetector

// HighlightDetector is re-exported from highlight package for backward compatibility
type HighlightDetector = highlight.HighlightDetector

// EmbeddingExtractor is the active analysis primitive used by semantic segmentation
// and visual search. It produces one embedding vector per frame.
type EmbeddingExtractor interface {
	ExtractFeatures(frame image.Image) ([]float32, error)
	Name() string
	Close() error
}
