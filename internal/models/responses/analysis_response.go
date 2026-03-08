package responses

import (
	"github.com/srad/mediasink/internal/db"
)

type AnalysisResponse struct {
	AnalysisID  *uint                    `json:"analysisId" extensions:"!x-nullable"`
	RecordingID uint                     `json:"recordingId" extensions:"!x-nullable"`
	Status      *string                  `json:"status" extensions:"!x-nullable"`
	Scenes      []db.SceneInfo     `json:"scenes"`
	Highlights  []db.HighlightInfo `json:"highlights"`
}

// NewAnalysisResponse converts a VideoAnalysisResult to an API response
// Returns a response with recordingId and nil values if result is nil (not analyzed yet)
func NewAnalysisResponse(recordingID uint, result *db.VideoAnalysisResult) (*AnalysisResponse, error) {
	// If no analysis record exists, return response with recordingId and nil values
	if result == nil {
		return &AnalysisResponse{
			RecordingID: recordingID,
			Scenes:      []db.SceneInfo{},
			Highlights:  []db.HighlightInfo{},
		}, nil
	}

	scenes, err := result.GetScenes()
	if err != nil {
		return nil, err
	}

	highlights, err := result.GetHighlights()
	if err != nil {
		return nil, err
	}

	if scenes == nil {
		scenes = []db.SceneInfo{}
	}
	if highlights == nil {
		highlights = []db.HighlightInfo{}
	}

	analysisID := result.AnalysisID
	status := string(result.Status)

	return &AnalysisResponse{
		AnalysisID:  &analysisID,
		RecordingID: uint(result.RecordingID),
		Status:      &status,
		Scenes:      scenes,
		Highlights:  highlights,
	}, nil
}
