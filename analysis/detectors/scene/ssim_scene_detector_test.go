package scene

import (
	"image"
	"image/color"
	"testing"

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

func TestSSIMSceneDetector_IdenticalFrames(t *testing.T) {
	detector := NewSSIMSceneDetector()

	// Create two identical frames
	img1 := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	img2 := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	frames := []image.Image{img1, img2}
	timestamps := []float64{0.0, 1.0}

	scenes, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Fatalf("DetectScenes failed: %v", err)
	}

	// Identical frames should produce 1 scene (the final scene), no scene changes
	if len(scenes) != 1 {
		t.Errorf("Expected 1 scene for identical frames, got %d", len(scenes))
	}
}

func TestSSIMSceneDetector_DifferentFrames(t *testing.T) {
	detector := NewSSIMSceneDetector()

	// Create many frames with clear scene changes to ensure meaningful detection
	frames := []image.Image{}
	timestamps := []float64{}

	// 5 frames of gray
	for i := 0; i < 5; i++ {
		frames = append(frames, createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 5 frames of black (scene change at frame 5)
	for i := 5; i < 10; i++ {
		frames = append(frames, createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	scenes, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Fatalf("DetectScenes failed: %v", err)
	}

	// With 10 frames and a clear scene change, we should get at least 2 scenes
	if len(scenes) < 2 {
		t.Errorf("Expected at least 2 scenes with clear scene change, got %d", len(scenes))
		for i, scene := range scenes {
			t.Logf("Scene %d: start=%.2f, end=%.2f, intensity=%.4f", i, scene.StartTime, scene.EndTime, scene.ChangeIntensity)
		}
	}
}

func TestSSIMSceneDetector_StatisticalMethod(t *testing.T) {
	// Create many frames with clear scene changes to test statistical method
	frames := []image.Image{}
	timestamps := []float64{}

	// 10 frames of gray
	for i := 0; i < 10; i++ {
		frames = append(frames, createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 10 frames of black (scene change at frame 10)
	for i := 10; i < 20; i++ {
		frames = append(frames, createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	// 10 frames of white (another scene change at frame 20)
	for i := 20; i < 30; i++ {
		frames = append(frames, createTestImage(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
		timestamps = append(timestamps, float64(i))
	}

	detector := NewSSIMSceneDetector() // Uses statistical method with k=5.0
	scenes, err := detector.DetectScenes(frames, timestamps)
	if err != nil {
		t.Fatalf("DetectScenes failed: %v", err)
	}

	// Should detect at least 2 scene changes (gray->black and black->white)
	if len(scenes) < 3 {
		t.Errorf("Expected at least 3 scenes with clear changes, got %d", len(scenes))
		for i, scene := range scenes {
			t.Logf("Scene %d: start=%.2f, end=%.2f, intensity=%.4f", i, scene.StartTime, scene.EndTime, scene.ChangeIntensity)
		}
	}

	t.Logf("Statistical method (k=2.5) detected %d scenes from %d frames", len(scenes), len(frames))
}

func TestCalculateSSIM(t *testing.T) {
	// Test SSIM calculation
	img1 := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})
	img2 := createTestImage(100, 100, color.RGBA{R: 128, G: 128, B: 128, A: 255})

	ssim := metrics.CalculateSSIM(img1, img2)

	t.Logf("SSIM of identical images: %.4f", ssim)

	// Identical images should have SSIM close to 1.0
	if ssim < 0.99 {
		t.Errorf("Expected SSIM close to 1.0 for identical images, got %.4f", ssim)
	}

	// Test with different images
	img3 := createTestImage(100, 100, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	img4 := createTestImage(100, 100, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	ssimDiff := metrics.CalculateSSIM(img3, img4)
	t.Logf("SSIM of black vs white images: %.4f", ssimDiff)

	// Very different images should have low SSIM
	if ssimDiff > 0.5 {
		t.Errorf("Expected low SSIM for very different images, got %.4f", ssimDiff)
	}
}
