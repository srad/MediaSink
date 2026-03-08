package scene

import (
	"image"

	"github.com/srad/mediasink/internal/db"
)

// SceneDetector defines the interface for scene detection algorithms
type SceneDetector interface {
	// DetectScenes analyzes frames to detect scene boundaries
	// Returns scenes with start/end times and change intensity
	DetectScenes(frames []image.Image, timestamps []float64) ([]db.SceneInfo, error)

	// Name returns the name of the detector algorithm
	Name() string

	// Close releases any resources held by the detector
	Close() error
}
