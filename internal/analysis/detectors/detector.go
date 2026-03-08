package detectors

import (
	"github.com/srad/mediasink/internal/analysis/detectors/highlight"
	"github.com/srad/mediasink/internal/analysis/detectors/scene"
)

// SceneDetector is re-exported from scene package for backward compatibility
type SceneDetector = scene.SceneDetector

// HighlightDetector is re-exported from highlight package for backward compatibility
type HighlightDetector = highlight.HighlightDetector
