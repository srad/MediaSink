package responses

import (
	"github.com/srad/mediasink/database"
)

type AnalysisResponse struct {
	AnalysisID  *uint                    `json:"analysisId" extensions:"!x-nullable"`
	RecordingID uint                     `json:"recordingId" extensions:"!x-nullable"`
	Status      *string                  `json:"status" extensions:"!x-nullable"`
	Scenes      []database.SceneInfo     `json:"scenes"`
	Highlights  []database.HighlightInfo `json:"highlights"`
}

// NewAnalysisResponse converts a VideoAnalysisResult to an API response
// Returns a response with recordingId and nil values if result is nil (not analyzed yet)
func NewAnalysisResponse(recordingID uint, result *database.VideoAnalysisResult) (*AnalysisResponse, error) {
	// If no analysis record exists, return response with recordingId and nil values
	if result == nil {
		return &AnalysisResponse{
			RecordingID: recordingID,
			Scenes:      []database.SceneInfo{},
			Highlights:  []database.HighlightInfo{},
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
		scenes = []database.SceneInfo{}
	}
	if highlights == nil {
		highlights = []database.HighlightInfo{}
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
