package services

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/srad/mediasink/database"
)

// ----- helpers ---------------------------------------------------------------

func writeTestFrame(dir string, timestamp int, c color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(filepath.Join(dir, fmt.Sprintf("%d.jpg", timestamp)))
	if err != nil {
		return err
	}
	defer f.Close()
	return jpeg.Encode(f, img, nil)
}

// ----- getPreviewFramePaths --------------------------------------------------

func TestGetPreviewFramePaths_SortsNumerically(t *testing.T) {
	dir := t.TempDir()
	// Write frames out-of-order to confirm numeric (not lexicographic) sort.
	for _, ts := range []int{100, 0, 10, 20} {
		if err := writeTestFrame(dir, ts, color.RGBA{R: 128, A: 255}); err != nil {
			t.Fatalf("writeTestFrame: %v", err)
		}
	}

	paths, timestamps, err := getPreviewFramePaths(dir)
	if err != nil {
		t.Fatalf("getPreviewFramePaths: %v", err)
	}

	want := []float64{0, 10, 20, 100}
	if len(timestamps) != len(want) {
		t.Fatalf("expected %d timestamps, got %d", len(want), len(timestamps))
	}
	for i, w := range want {
		if timestamps[i] != w {
			t.Errorf("timestamps[%d]: want %.0f, got %.0f", i, w, timestamps[i])
		}
		_ = paths[i]
	}
}

func TestGetPreviewFramePaths_SkipsNonJpeg(t *testing.T) {
	dir := t.TempDir()
	writeTestFrame(dir, 0, color.RGBA{R: 1, A: 255})
	writeTestFrame(dir, 5, color.RGBA{R: 2, A: 255})
	os.WriteFile(filepath.Join(dir, "README.txt"), []byte("skip me"), 0644)
	os.WriteFile(filepath.Join(dir, "abc.jpg"), []byte("not a number"), 0644)

	_, timestamps, err := getPreviewFramePaths(dir)
	if err != nil {
		t.Fatalf("getPreviewFramePaths: %v", err)
	}
	if len(timestamps) != 2 {
		t.Errorf("expected 2 valid frames, got %d", len(timestamps))
	}
}

func TestGetPreviewFramePaths_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	paths, timestamps, err := getPreviewFramePaths(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 0 || len(timestamps) != 0 {
		t.Errorf("expected empty result for empty directory")
	}
}

func TestGetPreviewFramePaths_InvalidFormatInNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frame-000001.jpg"), []byte("legacy"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, _, err := getPreviewFramePaths(dir)
	if err == nil {
		t.Fatalf("expected error for non-empty directory without timestamp files")
	}
	if !strings.Contains(err.Error(), "invalid preview frame format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ----- detectScenesFromSimilarities ------------------------------------------

// makeSimSeq builds a similarity sequence of n frames at highSim with low-sim
// "cuts" spanning indices [cutStart, cutEnd).
//
// Requirements so that a cut is actually detected:
//   - Span >= 3 consecutive frames so median smoothing (window=3) preserves the dip.
//   - n/m > k² where n=high-count, m=cut-count and k is the statistical threshold
//     multiplier (2.0 for scenes, 2.5 for highlights).  E.g. for k=2.0 use
//     ≥ 5 cuts in ≥ 30 total values.
//   - highSim and lowSim well separated (e.g. 0.85 vs 0.20) so the computed
//     threshold sits strictly above lowSim.
func makeSimSeq(n int, highSim float64, cutStart, cutEnd int, lowSim float64) ([]float64, []float64) {
	sims := make([]float64, n)
	ts := make([]float64, n)
	for i := range sims {
		ts[i] = float64(i + 1)
		if i >= cutStart && i < cutEnd {
			sims[i] = lowSim
		} else {
			sims[i] = highSim
		}
	}
	return sims, ts
}

func TestDetectScenesFromSimilarities_DetectsCut(t *testing.T) {
	// 30 frames: 25 high-similarity (0.85), 5 consecutive cuts (0.20).
	// n/m = 25/5 = 5 > k²=4 → threshold sits above 0.20 → cut is detected.
	sims, ts := makeSimSeq(30, 0.85, 12, 17, 0.20)

	scenes, err := detectScenesFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectScenesFromSimilarities: %v", err)
	}
	if len(scenes) < 2 {
		t.Errorf("expected at least 2 scenes for a clear multi-frame cut, got %d", len(scenes))
	}
}

func TestDetectScenesFromSimilarities_AllIdentical(t *testing.T) {
	sims, ts := makeSimSeq(10, 1.0, 0, 0, 0) // no cut

	scenes, err := detectScenesFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectScenesFromSimilarities: %v", err)
	}
	// All identical → no boundaries detected → exactly 1 final catch-all scene.
	if len(scenes) != 1 {
		t.Errorf("expected 1 scene for all-identical frames, got %d", len(scenes))
	}
}

func TestDetectScenesFromSimilarities_StartEndTimes(t *testing.T) {
	// First scene runs from 0 to the first cut timestamp; last scene ends at
	// the last timestamp in the sequence.  30 values, 5 cuts → n/m=5 > k²=4.
	sims, ts := makeSimSeq(30, 0.85, 5, 10, 0.20)

	scenes, err := detectScenesFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectScenesFromSimilarities: %v", err)
	}
	if len(scenes) < 2 {
		t.Fatalf("expected at least 2 scenes, got %d", len(scenes))
	}

	first := scenes[0]
	if first.StartTime != 0.0 {
		t.Errorf("first scene StartTime: want 0.0, got %.1f", first.StartTime)
	}

	last := scenes[len(scenes)-1]
	if last.EndTime != ts[len(ts)-1] {
		t.Errorf("last scene EndTime: want %.1f, got %.1f", ts[len(ts)-1], last.EndTime)
	}
}

func TestDetectScenesFromSimilarities_ChangeIntensity(t *testing.T) {
	// All detected scenes must have intensity in [0, 1].
	sims, ts := makeSimSeq(30, 0.85, 4, 9, 0.20)

	scenes, err := detectScenesFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectScenesFromSimilarities: %v", err)
	}
	for _, s := range scenes {
		if s.ChangeIntensity > 1.0 || s.ChangeIntensity < 0 {
			t.Errorf("ChangeIntensity out of range [0,1]: %.4f", s.ChangeIntensity)
		}
	}
}

// ----- detectHighlightsFromSimilarities --------------------------------------

func TestDetectHighlightsFromSimilarities_DetectsMotion(t *testing.T) {
	// 50 frames: 46 at 0.85, one motion event of 4 consecutive frames at 0.20.
	// n/m = 46/4 = 11.5 > k²=6.25 → threshold sits above 0.20 → event detected.
	sims, ts := makeSimSeq(50, 0.85, 23, 27, 0.20)

	highlights, err := detectHighlightsFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectHighlightsFromSimilarities: %v", err)
	}
	if len(highlights) == 0 {
		t.Error("expected highlights for low-similarity frames")
	}
	for _, h := range highlights {
		if h.Type != "motion" {
			t.Errorf("unexpected highlight type: %s", h.Type)
		}
		if h.Intensity <= 0 || h.Intensity > 1.0 {
			t.Errorf("highlight intensity out of range: %.4f", h.Intensity)
		}
	}
}

func TestDetectHighlightsFromSimilarities_AllIdentical(t *testing.T) {
	sims, ts := makeSimSeq(10, 1.0, 0, 0, 0)

	highlights, err := detectHighlightsFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectHighlightsFromSimilarities: %v", err)
	}
	if len(highlights) != 0 {
		t.Errorf("expected 0 highlights for identical frames, got %d", len(highlights))
	}
}

func TestDetectHighlightsFromSimilarities_TooFew(t *testing.T) {
	highlights, err := detectHighlightsFromSimilarities([]float64{0.5}, []float64{1.0}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if highlights != nil {
		t.Errorf("expected nil for fewer than 2 similarities, got %v", highlights)
	}
}

func TestDetectHighlightsFromSimilarities_Intensity(t *testing.T) {
	// Intensity = 1 - smoothed_similarity; for sim=0.20 intensity should be ~0.80.
	// 50 frames, 4 cuts at 0.20: n/m=46/4=11.5 > k²=6.25 → threshold > 0.20.
	sims, ts := makeSimSeq(50, 0.85, 5, 9, 0.20)

	highlights, err := detectHighlightsFromSimilarities(sims, ts, nil)
	if err != nil {
		t.Fatalf("detectHighlightsFromSimilarities: %v", err)
	}
	for _, h := range highlights {
		if h.Intensity <= 0 || h.Intensity > 1.0 {
			t.Errorf("intensity out of range [0,1]: %.4f", h.Intensity)
		}
	}
}

// ----- FeatureExtractor interface --------------------------------------------

func TestFeatureExtractorInterface_Signature(t *testing.T) {
	// Compile-time check: the interface requires []float32 return.
	var _ FeatureExtractor = (interface {
		ExtractFeatures(image.Image) ([]float32, error)
	})(nil)
}

// ----- database.RecordingID type sanity ------------------------------------

func TestRecordingID_Type(t *testing.T) {
	id := database.RecordingID(42)
	if uint(id) != 42 {
		t.Errorf("expected 42, got %d", id)
	}
}

// ----- loadFrame -------------------------------------------------------------

func TestLoadFrame_ValidJpeg(t *testing.T) {
	dir := t.TempDir()
	if err := writeTestFrame(dir, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255}); err != nil {
		t.Fatalf("writeTestFrame: %v", err)
	}
	img, err := loadFrame(filepath.Join(dir, "0.jpg"))
	if err != nil {
		t.Fatalf("loadFrame: %v", err)
	}
	if img.Bounds().Dx() == 0 {
		t.Error("expected non-zero image width")
	}
}

func TestLoadFrame_Missing(t *testing.T) {
	_, err := loadFrame("/nonexistent/path/frame.jpg")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
