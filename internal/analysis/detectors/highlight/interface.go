package highlight

import (
	"image"

	"github.com/srad/mediasink/internal/db"
)

// HighlightDetector defines the interface for highlight detection algorithms
type HighlightDetector interface {
	// DetectHighlights analyzes frames to detect moments with activity/motion
	// Returns highlights with timestamps and intensity values
	DetectHighlights(frames []image.Image, timestamps []float64) ([]db.HighlightInfo, error)

	// Name returns the name of the detector algorithm
	Name() string

	// Close releases any resources held by the detector
	Close() error
}
