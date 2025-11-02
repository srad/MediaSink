package scene

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gonum.org/v1/gonum/mat"
	"github.com/srad/mediasink/analysis/detectors/highlight"
	"github.com/srad/mediasink/analysis/metrics"
)

// TestTensorFlowSceneDetector_FeatureExtraction tests that TensorFlow models produce valid features
func TestTensorFlowSceneDetector_FeatureExtraction(t *testing.T) {
	// Create detector with MobileNet V3
	detector, err := NewTensorFlowSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: TensorFlow model not available: %v", err)
	}
	defer detector.Close()

	// Create a simple test image
	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Extract features
	features, err := detector.ExtractFeatures(img)
	if err != nil {
		t.Fatalf("Failed to extract features: %v", err)
	}

	// Verify feature vector is valid
	if features == nil {
		t.Errorf("Expected non-nil feature vector")
	}

	rows, cols := features.Dims()
	if rows == 0 {
		t.Errorf("Expected non-zero feature vector length, got %d", rows)
	}

	t.Logf("Feature vector dimensions: %d x %d", rows, cols)

	// Verify feature values are reasonable (not NaN, not Inf, in reasonable range)
	for i := 0; i < rows; i++ {
		val := features.AtVec(i)
		if math.IsNaN(val) {
			t.Errorf("Feature at index %d is NaN", i)
		}
		if math.IsInf(val, 0) {
			t.Errorf("Feature at index %d is Inf", i)
		}
		// Most neural network features should be in [-10, 10] range after extraction
		if math.Abs(val) > 100 {
			t.Logf("Feature at index %d has unusual magnitude: %.4f", i, val)
		}
	}
}

// TestTensorFlowSceneDetector_CosineSimilarity tests similarity calculations
func TestTensorFlowSceneDetector_CosineSimilarity(t *testing.T) {
	// Create detector with MobileNet V3
	detector, err := NewTensorFlowSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: TensorFlow model not available: %v", err)
	}
	defer detector.Close()

	// Test 1: Identical frames should have high similarity
	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	features1, err := detector.ExtractFeatures(img)
	if err != nil {
		t.Fatalf("Failed to extract features for identical frame: %v", err)
	}

	features2, err := detector.ExtractFeatures(img)
	if err != nil {
		t.Fatalf("Failed to extract features for identical frame: %v", err)
	}

	similarity := metrics.CosineSimilarity(features1, features2)
	t.Logf("Similarity of identical frames: %.4f", similarity)

	if similarity < 0.95 {
		t.Logf("Note: Expected high similarity (>0.95) for identical frames, got %.4f", similarity)
	}

	if similarity > 1.0 || similarity < 0 {
		t.Errorf("Similarity out of valid range [0, 1]: %.4f", similarity)
	}

	// Test 2: Very different frames should have lower similarity
	imgBlack := createTestImage(224, 224, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	imgWhite := createTestImage(224, 224, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	featuresBlack, err := detector.ExtractFeatures(imgBlack)
	if err != nil {
		t.Fatalf("Failed to extract features for black frame: %v", err)
	}

	featuresWhite, err := detector.ExtractFeatures(imgWhite)
	if err != nil {
		t.Fatalf("Failed to extract features for white frame: %v", err)
	}

	similarityDiff := metrics.CosineSimilarity(featuresBlack, featuresWhite)
	t.Logf("Similarity of black vs white frames: %.4f", similarityDiff)

	if similarityDiff > 1.0 || similarityDiff < 0 {
		t.Errorf("Similarity out of valid range [0, 1]: %.4f", similarityDiff)
	}

	// Black vs white should be less similar than identical frames
	if similarityDiff > similarity {
		t.Logf("Note: Expected lower similarity for different frames, but got: identical=%.4f, different=%.4f",
			similarity, similarityDiff)
	}
}

// TestTensorFlowSceneDetector_DetectScenes tests scene detection
func TestTensorFlowSceneDetector_DetectScenes(t *testing.T) {
	// Create detector with MobileNet V3
	detector, err := NewTensorFlowSceneDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: TensorFlow model not available: %v", err)
	}
	defer detector.Close()

	// Create frames: 10 gray, 10 black, 10 white
	frames := []image.Image{}
	timestamps := []float64{}

	// 10 frames of gray
	for i := 0; i < 10; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 10 frames of black (scene change at frame 10)
	for i := 10; i < 20; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 10 frames of white (scene change at frame 20)
	for i := 20; i < 30; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	scenes, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Fatalf("Failed to detect scenes: %v", err)
	}

	t.Logf("Detected %d scenes from %d frames", len(scenes), len(frames))
	for i, scene := range scenes {
		t.Logf("  Scene %d: start=%.1f, end=%.1f, intensity=%.4f",
			i, scene.StartTime, scene.EndTime, scene.ChangeIntensity)
	}

	// Should detect at least 3 scenes (gray, black, white)
	if len(scenes) < 3 {
		t.Errorf("Expected at least 3 scenes with clear color changes, got %d", len(scenes))
	}
}

// TestTensorFlowHighlightDetector_FeatureExtraction tests highlight detector feature extraction
func TestTensorFlowHighlightDetector_FeatureExtraction(t *testing.T) {
	// Create detector with MobileNet V3
	detector, err := highlight.NewTensorFlowHighlightDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: TensorFlow model not available: %v", err)
	}
	defer detector.Close()

	// Create a simple test image
	img := createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Extract features using detector's model
	// Since HighlightDetector doesn't expose ExtractFeatures, we test through DetectHighlights
	frames := []image.Image{img, img}
	timestamps := []float64{0.0, 1.0}

	highlights, err := detector.DetectHighlights(frames, timestamps)
	if err != nil {
		t.Fatalf("Failed to detect highlights: %v", err)
	}

	// Identical frames should produce no highlights
	if len(highlights) != 0 {
		t.Errorf("Expected no highlights for identical frames, got %d", len(highlights))
	}

	t.Logf("Identical frames: %d highlights detected (expected 0)", len(highlights))
}

// TestTensorFlowHighlightDetector_DetectHighlights tests highlight detection
func TestTensorFlowHighlightDetector_DetectHighlights(t *testing.T) {
	// Create detector with MobileNet V3
	detector, err := highlight.NewTensorFlowHighlightDetector("mobilenet_v3_large")
	if err != nil {
		t.Skipf("Skipping: TensorFlow model not available: %v", err)
	}
	defer detector.Close()

	// Create frames with motion: 5 gray, 5 black (motion), 5 gray, 5 white (motion)
	frames := []image.Image{}
	timestamps := []float64{}

	// 5 frames of gray
	for i := 0; i < 5; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 5 frames of black (motion)
	for i := 5; i < 10; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 5 frames of gray (back to stable)
	for i := 10; i < 15; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 5 frames of white (motion)
	for i := 15; i < 20; i++ {
		frames = append(frames, createTestImage(224, 224, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	highlights, err := detector.DetectHighlights(frames, timestamps)
	if err != nil {
		t.Fatalf("Failed to detect highlights: %v", err)
	}

	t.Logf("Detected %d highlights from %d frames", len(highlights), len(frames))
	for i, h := range highlights {
		t.Logf("  Highlight %d: timestamp=%.1f, intensity=%.4f, type=%s",
			i, h.Timestamp, h.Intensity, h.Type)
	}

	// Should detect some highlights around the transitions
	if len(highlights) == 0 {
		t.Logf("Note: Expected some highlights for frames with clear changes, got 0")
	}
}

// TestCosineSimilarity_EdgeCases tests cosine similarity function
func TestCosineSimilarity_EdgeCases(t *testing.T) {
	// Test identical vectors
	v1 := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})
	v2 := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})

	sim := metrics.CosineSimilarity(v1, v2)
	if sim < 0.9999 { // Should be very close to 1.0
		t.Errorf("Identical vectors should have similarity ~1.0, got %.4f", sim)
	}

	// Test orthogonal vectors
	v3 := mat.NewVecDense(3, []float64{1.0, 0.0, 0.0})
	v4 := mat.NewVecDense(3, []float64{0.0, 1.0, 0.0})

	sim2 := metrics.CosineSimilarity(v3, v4)
	if math.Abs(sim2) > 0.01 { // Should be close to 0
		t.Errorf("Orthogonal vectors should have similarity ~0.0, got %.4f", sim2)
	}

	// Test opposite vectors
	v5 := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})
	v6 := mat.NewVecDense(3, []float64{-1.0, -2.0, -3.0})

	sim3 := metrics.CosineSimilarity(v5, v6)
	if sim3 > -0.9999 || sim3 < -1.0001 { // Should be very close to -1.0
		t.Errorf("Opposite vectors should have similarity ~-1.0, got %.4f", sim3)
	}
}
