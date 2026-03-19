package db

import "testing"

func TestVideoAnalysisResult_SetAndGetSegments(t *testing.T) {
	result := &VideoAnalysisResult{}
	want := []SegmentInfo{
		{
			Kind:                    "chapter",
			StartTime:               0,
			EndTime:                 120,
			Confidence:              0.8,
			RepresentativeTimestamp: 60,
		},
		{
			Kind:                    "chapter",
			StartTime:               120,
			EndTime:                 300,
			Confidence:              0.6,
			RepresentativeTimestamp: 210,
		},
	}

	if err := result.SetSegments(want); err != nil {
		t.Fatalf("SetSegments: %v", err)
	}

	got, err := result.GetSegments()
	if err != nil {
		t.Fatalf("GetSegments: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d segments, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("segment[%d]: want %+v, got %+v", i, want[i], got[i])
		}
	}
}

func TestVideoAnalysisResult_GetSegments_Empty(t *testing.T) {
	result := &VideoAnalysisResult{}

	got, err := result.GetSegments()
	if err != nil {
		t.Fatalf("GetSegments: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty segment list, got %+v", got)
	}
}

func TestVideoAnalysisResult_SetAndGetHighlights_WithRanges(t *testing.T) {
	result := &VideoAnalysisResult{}
	want := []HighlightInfo{
		{
			StartTime: 12,
			EndTime:   24,
			Timestamp: 18,
			Intensity: 0.7,
			Type:      "motion",
		},
	}

	if err := result.SetHighlights(want); err != nil {
		t.Fatalf("SetHighlights: %v", err)
	}

	got, err := result.GetHighlights()
	if err != nil {
		t.Fatalf("GetHighlights: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d highlights, got %d", len(want), len(got))
	}
	if got[0] != want[0] {
		t.Fatalf("highlight: want %+v, got %+v", want[0], got[0])
	}
}
