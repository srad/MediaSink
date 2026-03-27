package responses

import (
	"testing"

	"github.com/srad/mediasink/server/internal/db"
)

func TestNewAnalysisResponse_NilResultIncludesEmptySegments(t *testing.T) {
	response, err := NewAnalysisResponse(42, nil)
	if err != nil {
		t.Fatalf("NewAnalysisResponse: %v", err)
	}

	if response.RecordingID != 42 {
		t.Fatalf("expected recording ID 42, got %d", response.RecordingID)
	}
	if len(response.Scenes) != 0 {
		t.Fatalf("expected empty scenes, got %+v", response.Scenes)
	}
	if len(response.Highlights) != 0 {
		t.Fatalf("expected empty highlights, got %+v", response.Highlights)
	}
	if len(response.Segments) != 0 {
		t.Fatalf("expected empty segments, got %+v", response.Segments)
	}
}

func TestNewAnalysisResponse_PopulatesSegments(t *testing.T) {
	analysis := &db.VideoAnalysisResult{
		AnalysisID:  7,
		RecordingID: db.RecordingID(42),
		Status:      db.AnalysisCompleted,
	}

	scenes := []db.SceneInfo{{StartTime: 0, EndTime: 120, ChangeIntensity: 0.7}}
	highlights := []db.HighlightInfo{{StartTime: 30, EndTime: 40, Timestamp: 33, Intensity: 0.5, Type: "motion"}}
	segments := []db.SegmentInfo{{
		Kind:                    "chapter",
		StartTime:               0,
		EndTime:                 120,
		Confidence:              0.7,
		RepresentativeTimestamp: 60,
	}}

	if err := analysis.SetScenes(scenes); err != nil {
		t.Fatalf("SetScenes: %v", err)
	}
	if err := analysis.SetHighlights(highlights); err != nil {
		t.Fatalf("SetHighlights: %v", err)
	}
	if err := analysis.SetSegments(segments); err != nil {
		t.Fatalf("SetSegments: %v", err)
	}

	response, err := NewAnalysisResponse(42, analysis)
	if err != nil {
		t.Fatalf("NewAnalysisResponse: %v", err)
	}

	if response.AnalysisID == nil || *response.AnalysisID != 7 {
		t.Fatalf("unexpected analysis ID: %+v", response.AnalysisID)
	}
	if response.Status == nil || *response.Status != string(db.AnalysisCompleted) {
		t.Fatalf("unexpected status: %+v", response.Status)
	}
	if len(response.Scenes) != 1 || response.Scenes[0] != scenes[0] {
		t.Fatalf("unexpected scenes: %+v", response.Scenes)
	}
	if len(response.Highlights) != 1 || response.Highlights[0] != highlights[0] {
		t.Fatalf("unexpected highlights: %+v", response.Highlights)
	}
	if len(response.Segments) != 1 || response.Segments[0] != segments[0] {
		t.Fatalf("unexpected segments: %+v", response.Segments)
	}
}
