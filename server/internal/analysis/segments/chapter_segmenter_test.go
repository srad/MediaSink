package segments

import "testing"

type fixtureSegment struct {
	duration float64
	values   []float32
}

func makeFrames(step float64, defs ...fixtureSegment) ([]EmbeddingFrame, float64) {
	frames := make([]EmbeddingFrame, 0)
	timestamp := 0.0
	for _, def := range defs {
		count := int(def.duration / step)
		for i := 0; i < count; i++ {
			values := append([]float32(nil), def.values...)
			frames = append(frames, EmbeddingFrame{
				Timestamp: timestamp,
				Values:    values,
			})
			timestamp += step
		}
	}
	return frames, timestamp
}

func TestDetectChapters_DetectsSemanticBoundary(t *testing.T) {
	frames, totalDuration := makeFrames(10,
		fixtureSegment{duration: 150, values: []float32{1, 0}},
		fixtureSegment{duration: 150, values: []float32{0, 1}},
	)

	segments, err := DetectChapters(frames, totalDuration, DefaultChapterConfig())
	if err != nil {
		t.Fatalf("DetectChapters: %v", err)
	}
	if len(segments) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(segments))
	}
	if segments[0].Kind != chapterKind || segments[1].Kind != chapterKind {
		t.Fatalf("expected chapter kinds, got %+v", segments)
	}
	if segments[0].EndTime < 120 || segments[0].EndTime > 180 {
		t.Fatalf("expected split near the midpoint, got first chapter end %.1f", segments[0].EndTime)
	}
	if segments[1].StartTime != segments[0].EndTime {
		t.Fatalf("expected contiguous segments, got %.1f -> %.1f", segments[0].EndTime, segments[1].StartTime)
	}
}

func TestDetectChapters_NoBoundaryReturnsSingleSegment(t *testing.T) {
	frames, totalDuration := makeFrames(10,
		fixtureSegment{duration: 300, values: []float32{1, 0}},
	)

	segments, err := DetectChapters(frames, totalDuration, DefaultChapterConfig())
	if err != nil {
		t.Fatalf("DetectChapters: %v", err)
	}
	if len(segments) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(segments))
	}
	if segments[0].StartTime != 0 || segments[0].EndTime != totalDuration {
		t.Fatalf("unexpected segment bounds: %+v", segments[0])
	}
}

func TestMergeShortChapterSegments_PrefersCloserNeighbor(t *testing.T) {
	frames, totalDuration := makeFrames(10,
		fixtureSegment{duration: 120, values: []float32{1, 0}},
		fixtureSegment{duration: 30, values: []float32{0.9, 0.1}},
		fixtureSegment{duration: 150, values: []float32{0, 1}},
	)

	splits := mergeShortChapterSegments(frames, totalDuration, []int{12, 15}, DefaultChapterConfig())
	if len(splits) != 1 {
		t.Fatalf("expected 1 split after merge, got %v", splits)
	}
	if splits[0] != 15 {
		t.Fatalf("expected the short middle segment to merge into the previous segment, got split %v", splits)
	}
}

func TestSplitLongChapterSegments_UsesStrongestRemainingPeak(t *testing.T) {
	frames, totalDuration := makeFrames(10,
		fixtureSegment{duration: 900, values: []float32{1, 0}},
		fixtureSegment{duration: 900, values: []float32{0.6, 0.4}},
	)

	config := DefaultChapterConfig()
	candidates := []boundaryCandidate{
		{SplitIndex: 30, Timestamp: 300, Score: 0.2, Confidence: 0.2, Peak: true},
		{SplitIndex: 90, Timestamp: 900, Score: 0.8, Confidence: 0.9, Peak: true},
		{SplitIndex: 150, Timestamp: 1500, Score: 0.5, Confidence: 0.5, Peak: true},
	}

	splits := splitLongChapterSegments(frames, totalDuration, nil, candidates, config)
	if len(splits) != 1 {
		t.Fatalf("expected one inserted split, got %v", splits)
	}
	if splits[0] != 90 {
		t.Fatalf("expected strongest valid peak at split 90, got %v", splits)
	}
}

func TestSplitLongChapterSegments_IgnoresZeroConfidencePeaks(t *testing.T) {
	frames, totalDuration := makeFrames(10,
		fixtureSegment{duration: 900, values: []float32{1, 0}},
		fixtureSegment{duration: 900, values: []float32{0.8, 0.2}},
	)

	config := DefaultChapterConfig()
	candidates := []boundaryCandidate{
		{SplitIndex: 60, Timestamp: 600, Score: 0.9, Confidence: 0, Peak: true},
		{SplitIndex: 120, Timestamp: 1200, Score: 0.85, Confidence: 0, Peak: true},
	}

	splits := splitLongChapterSegments(frames, totalDuration, nil, candidates, config)
	if len(splits) != 0 {
		t.Fatalf("expected no inserted split for zero-confidence peaks, got %v", splits)
	}
}

func TestSelectPrimarySplits_RespectsMinimumPeakScore(t *testing.T) {
	splits := selectPrimarySplits([]boundaryCandidate{
		{SplitIndex: 10, Score: 0.1, Confidence: 0.8, Peak: true},
		{SplitIndex: 20, Score: 0.2, Confidence: 0.7, Peak: true},
	}, DefaultChapterConfig().MinPeakScore)

	if len(splits) != 1 || splits[0] != 20 {
		t.Fatalf("expected only score-above-floor split, got %v", splits)
	}
}
