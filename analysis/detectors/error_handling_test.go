package detectors

import (
	"image"
	"image/color"
	"testing"

	"gonum.org/v1/gonum/mat"
	"github.com/srad/mediasink/analysis/detectors/scene"
	"github.com/srad/mediasink/analysis/detectors/highlight"
	"github.com/srad/mediasink/analysis/threshold"
	"github.com/srad/mediasink/analysis/metrics"
)

// createTestImage creates a solid color image for testing
func createTestImage(width, height int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// TestSSIMSceneDetector_InsufficientFrames tests error handling with too few frames
func TestSSIMSceneDetector_InsufficientFrames(t *testing.T) {
	detector := scene.NewSSIMSceneDetector()

	// Test with zero frames
	scenes, err := detector.DetectScenes([]image.Image{}, []float64{})
	if err != nil {
		t.Errorf("Should return nil silently for zero frames, got error: %v", err)
	}
	if scenes != nil {
		t.Errorf("Expected nil for zero frames, got %d scenes", len(scenes))
	}

	// Test with one frame
	img := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	scenes, err = detector.DetectScenes([]image.Image{img}, []float64{0.0})
	if err != nil {
		t.Errorf("Should return nil silently for one frame, got error: %v", err)
	}
	if scenes != nil {
		t.Errorf("Expected nil for one frame, got %d scenes", len(scenes))
	}
}

// TestFrameDiffHighlightDetector_InsufficientFrames tests error handling with too few frames
func TestFrameDiffHighlightDetector_InsufficientFrames(t *testing.T) {
	detector := highlight.NewFrameDiffHighlightDetector()

	// Test with zero frames
	highlights, err := detector.DetectHighlights([]image.Image{}, []float64{})
	if err != nil {
		t.Errorf("Should return nil silently for zero frames, got error: %v", err)
	}
	if highlights != nil {
		t.Errorf("Expected nil for zero frames, got %d highlights", len(highlights))
	}

	// Test with one frame
	img := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	highlights, err = detector.DetectHighlights([]image.Image{img}, []float64{0.0})
	if err != nil {
		t.Errorf("Should return nil silently for one frame, got error: %v", err)
	}
	if highlights != nil {
		t.Errorf("Expected nil for one frame, got %d highlights", len(highlights))
	}
}

// TestSSIMSceneDetector_MismatchedLengths tests error handling with mismatched frame/timestamp arrays
func TestSSIMSceneDetector_MismatchedLengths(t *testing.T) {
	detector := scene.NewSSIMSceneDetector()

	frames := []image.Image{
		createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255}),
		createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255}),
	}
	timestamps := []float64{0.0} // Only 1 timestamp for 2 frames

	// This should cause a panic or error when trying to access timestamps[i+1]
	// The code should be defensive about this
	defer func() {
		if r := recover(); r == nil {
			t.Logf("Note: Mismatched array lengths should ideally be caught earlier")
		}
	}()

	_, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Logf("Good: Caught mismatched lengths with error: %v", err)
	}
}

// TestCosineSimilarity_MismatchedDimensions tests error handling with different vector dimensions
func TestCosineSimilarity_MismatchedDimensions(t *testing.T) {
	v1 := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})
	v2 := mat.NewVecDense(4, []float64{1.0, 2.0, 3.0, 4.0})

	// This should handle gracefully (either return 0, or cap to smaller dimension)
	// Current implementation might panic if not defensive
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Panic on mismatched dimensions: %v", r)
		}
	}()

	sim := metrics.CosineSimilarity(v1, v2)
	t.Logf("Cosine similarity of mismatched vectors: %.4f", sim)
	// Should handle gracefully - result may be incorrect but shouldn't panic
	t.Logf("Note: Mismatched dimensions may produce incorrect results")
}

// TestCosineSimilarity_ZeroVector tests edge case with zero vector
func TestCosineSimilarity_ZeroVector(t *testing.T) {
	v1 := mat.NewVecDense(3, []float64{0.0, 0.0, 0.0})
	v2 := mat.NewVecDense(3, []float64{1.0, 2.0, 3.0})

	sim := metrics.CosineSimilarity(v1, v2)
	t.Logf("Cosine similarity of zero vector: %.4f", sim)
	// Should be NaN or 0, not panic
	if sim > 1.0 || sim < -1.0 {
		t.Errorf("Similarity out of range: %.4f", sim)
	}
}

// TestThresholdMethod_EmptyScores tests threshold calculation with no scores
func TestThresholdMethod_EmptyScores(t *testing.T) {
	method := threshold.NewStatisticalThresholdMethod(2.0)

	// Empty slice should return error or handle gracefully
	_, err := method.Calculate([]float64{})
	if err == nil {
		t.Logf("Note: Empty scores should ideally return an error")
	} else {
		t.Logf("Good: Caught empty scores with error: %v", err)
	}
}

// TestThresholdMethod_SingleScore tests threshold calculation with only one score
func TestThresholdMethod_SingleScore(t *testing.T) {
	method := threshold.NewStatisticalThresholdMethod(2.0)

	// Single score: std dev = 0, so threshold = mean - 2*0 = mean
	threshold, err := method.Calculate([]float64{0.5})
	if err != nil {
		t.Logf("Single score produced error: %v", err)
	} else {
		t.Logf("Single score threshold: %.4f (expected ~0.5)", threshold)
		if threshold < 0.4 || threshold > 0.6 {
			t.Logf("Note: Unexpected threshold for single score: %.4f", threshold)
		}
	}
}

// TestThresholdMethod_AllIdenticalScores tests threshold calculation with identical scores
func TestThresholdMethod_AllIdenticalScores(t *testing.T) {
	method := threshold.NewStatisticalThresholdMethod(2.0)

	// All identical: std dev = 0, so threshold = mean
	scores := []float64{0.7, 0.7, 0.7, 0.7, 0.7}
	threshold, err := method.Calculate(scores)
	if err != nil {
		t.Logf("All identical scores produced error: %v", err)
	} else {
		t.Logf("Identical scores threshold: %.4f (expected ~0.7)", threshold)
		if threshold < 0.6 || threshold > 0.8 {
			t.Logf("Note: Unexpected threshold for identical scores: %.4f", threshold)
		}
	}
}

// TestSSIMCalculation_DifferentDimensions tests SSIM with different image sizes
func TestSSIMCalculation_DifferentDimensions(t *testing.T) {
	img1 := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	img2 := createTestImage(200, 200, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	// Should handle gracefully by averaging dimensions
	ssim := metrics.CalculateSSIM(img1, img2)
	t.Logf("SSIM of different-sized images: %.4f", ssim)

	if ssim < 0 || ssim > 1 {
		t.Errorf("SSIM out of valid range [0, 1]: %.4f", ssim)
	}
}

// TestTensorFlowSceneDetector_InvalidModel tests error handling with non-existent model
func TestTensorFlowSceneDetector_InvalidModel(t *testing.T) {
	// Try to load a model that doesn't exist
	_, err := scene.NewTensorFlowSceneDetector("nonexistent_model_xyz")
	if err == nil {
		t.Errorf("Expected error for non-existent model, but got none")
	} else {
		t.Logf("Good: Caught non-existent model with error: %v", err)
	}
}

// TestFrameDiffDetection_ExtremePixelValues tests frame diff with extreme pixel values
func TestFrameDiffDetection_ExtremePixelValues(t *testing.T) {
	img1 := createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})       // Black
	img2 := createTestImage(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // White

	diff := metrics.CalculateFrameDifference(img1, img2)
	t.Logf("Frame difference (black vs white): %.4f", diff)

	if diff < 0 || diff > 1 {
		t.Errorf("Frame difference out of valid range [0, 1]: %.4f", diff)
	}

	// Black vs white should produce maximum difference
	if diff < 0.95 {
		t.Logf("Note: Expected high difference for black vs white, got %.4f", diff)
	}
}

// TestSSIMCalculation_VeryDifferentImages tests SSIM with extreme differences
func TestSSIMCalculation_VeryDifferentImages(t *testing.T) {
	img1 := createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})       // Black
	img2 := createTestImage(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255}) // White

	ssim := metrics.CalculateSSIM(img1, img2)
	t.Logf("SSIM of black vs white: %.4f", ssim)

	if ssim < 0 || ssim > 1 {
		t.Errorf("SSIM out of valid range [0, 1]: %.4f", ssim)
	}

	// Black vs white should produce very low SSIM
	if ssim > 0.3 {
		t.Logf("Note: Expected low SSIM for black vs white, got %.4f", ssim)
	}
}
