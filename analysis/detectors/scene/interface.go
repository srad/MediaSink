package scene

import (
	"image"

	"github.com/srad/mediasink/database"
	"gonum.org/v1/gonum/mat"
)

// SceneDetector defines the interface for scene detection algorithms
type SceneDetector interface {
	// DetectScenes analyzes frames to detect scene boundaries
	// Returns scenes with start/end times and change intensity
	DetectScenes(frames []image.Image, timestamps []float64) ([]database.SceneInfo, error)

	// Name returns the name of the detector algorithm
	Name() string

	// Close releases any resources held by the detector
	Close() error

	// ExtractFeatures extracts a feature vector from a frame
	ExtractFeatures(frame image.Image) (*mat.VecDense, error)
}
