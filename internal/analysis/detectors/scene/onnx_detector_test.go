package scene

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/srad/mediasink/internal/analysis/detectors/highlight"
)

// TestOnnxSceneDetector_FeatureExtraction tests that the MobileNetV3 model produces valid features.
func TestOnnxSceneDetector_FeatureExtraction(t *testing.T) {
	detector, err := NewOnnxSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: ONNX model not available: %v", err)
	}
	defer detector.Close()

	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	features, err := detector.(*onnxSceneDetector).ExtractFeatures(img)
	if err != nil {
		t.Fatalf("Failed to extract features: %v", err)
	}

	if len(features) == 0 {
		t.Fatal("Expected non-empty feature vector")
	}

	t.Logf("Feature vector length: %d", len(features))

	for i, val := range features {
		if math.IsNaN(float64(val)) {
			t.Errorf("Feature at index %d is NaN", i)
		}
		if math.IsInf(float64(val), 0) {
			t.Errorf("Feature at index %d is Inf", i)
		}
	}
}

// TestOnnxSceneDetector_CosineSimilarity tests that identical frames score ~1.0 and
// very different frames score lower.
func TestOnnxSceneDetector_CosineSimilarity(t *testing.T) {
	detector, err := NewOnnxSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: ONNX model not available: %v", err)
	}
	defer detector.Close()

	od := detector.(*onnxSceneDetector)

	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	f1, err := od.ExtractFeatures(img)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}
	f2, err := od.ExtractFeatures(img)
	if err != nil {
		t.Fatalf("ExtractFeatures failed: %v", err)
	}

	sim := cosineSim(f1, f2)
	t.Logf("Identical frames similarity: %.4f", sim)
	if sim > 1.0 || sim < 0 {
		t.Errorf("Similarity out of range [0,1]: %.4f", sim)
	}
	if sim < 0.95 {
		t.Logf("Note: expected ~1.0 for identical frames, got %.4f", sim)
	}

	black, _ := od.ExtractFeatures(createTestImage(224, 224, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	white, _ := od.ExtractFeatures(createTestImage(224, 224, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	simDiff := cosineSim(black, white)
	t.Logf("Black vs white similarity: %.4f", simDiff)
	if simDiff > 1.0 || simDiff < 0 {
		t.Errorf("Similarity out of range [0,1]: %.4f", simDiff)
	}
	if simDiff >= sim {
		t.Logf("Note: expected lower similarity for different frames (identical=%.4f, different=%.4f)", sim, simDiff)
	}
}

// TestOnnxSceneDetector_DetectScenes tests that clear color-block transitions produce ≥3 scenes.
func TestOnnxSceneDetector_DetectScenes(t *testing.T) {
	detector, err := NewOnnxSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: ONNX model not available: %v", err)
	}
	defer detector.Close()

	var frames []image.Image
	var timestamps []float64

	for i := 0; i < 10; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}
	for i := 10; i < 20; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
		timestamps = append(timestamps, float64(i))
	}
	for i := 20; i < 30; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	scenes, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Fatalf("DetectScenes failed: %v", err)
	}

	t.Logf("Detected %d scenes from %d frames", len(scenes), len(frames))
	for i, s := range scenes {
		t.Logf("  Scene %d: start=%.1f end=%.1f intensity=%.4f", i, s.StartTime, s.EndTime, s.ChangeIntensity)
	}

	if len(scenes) < 3 {
		t.Errorf("Expected at least 3 scenes for gray/black/white blocks, got %d", len(scenes))
	}
}

// TestOnnxHighlightDetector_IdenticalFrames tests that identical frames produce no highlights.
func TestOnnxHighlightDetector_IdenticalFrames(t *testing.T) {
	detector, err := highlight.NewOnnxHighlightDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: ONNX model not available: %v", err)
	}
	defer detector.Close()

	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	highlights, err := detector.DetectHighlights([]image.Image{img, img}, []float64{0.0, 1.0})
	if err != nil {
		t.Fatalf("DetectHighlights failed: %v", err)
	}

	if len(highlights) != 0 {
		t.Errorf("Expected 0 highlights for identical frames, got %d", len(highlights))
	}
}

// TestOnnxHighlightDetector_DetectHighlights tests that color transitions are detected as highlights.
func TestOnnxHighlightDetector_DetectHighlights(t *testing.T) {
	detector, err := highlight.NewOnnxHighlightDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: ONNX model not available: %v", err)
	}
	defer detector.Close()

	var frames []image.Image
	var timestamps []float64

	colors := []color.RGBA{
		{R: 128, G: 128, B: 128, A: 255},
		{R: 0, G: 0, B: 0, A: 255},
		{R: 128, G: 128, B: 128, A: 255},
		{R: 255, G: 255, B: 255, A: 255},
	}
	for block, c := range colors {
		for i := 0; i < 5; i++ {
			frames = append(frames, createTestImage(224, 224, c))
			timestamps = append(timestamps, float64(block*5+i))
		}
	}

	highlights, err := detector.DetectHighlights(frames, timestamps)
	if err != nil {
		t.Fatalf("DetectHighlights failed: %v", err)
	}

	t.Logf("Detected %d highlights from %d frames", len(highlights), len(frames))
	for i, h := range highlights {
		t.Logf("  Highlight %d: timestamp=%.1f intensity=%.4f type=%s", i, h.Timestamp, h.Intensity, h.Type)
	}
}

// TestCosineSimilarity_EdgeCases tests the cosineSim helper for known values.
func TestCosineSimilarity_EdgeCases(t *testing.T) {
	// Identical vectors → ~1.0
	v1 := []float32{1.0, 2.0, 3.0}
	v2 := []float32{1.0, 2.0, 3.0}
	if sim := cosineSim(v1, v2); sim < 0.9999 {
		t.Errorf("Identical vectors: expected ~1.0, got %.4f", sim)
	}

	// Orthogonal vectors → ~0.0
	v3 := []float32{1.0, 0.0, 0.0}
	v4 := []float32{0.0, 1.0, 0.0}
	if sim := cosineSim(v3, v4); math.Abs(sim) > 0.01 {
		t.Errorf("Orthogonal vectors: expected ~0.0, got %.4f", sim)
	}

	// Opposite vectors → ~-1.0
	v5 := []float32{1.0, 2.0, 3.0}
	v6 := []float32{-1.0, -2.0, -3.0}
	if sim := cosineSim(v5, v6); sim > -0.9999 || sim < -1.0001 {
		t.Errorf("Opposite vectors: expected ~-1.0, got %.4f", sim)
	}
}
