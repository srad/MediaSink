package preprocessing

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// solidImage creates a solid-color RGBA image.
func solidImage(w, h int, c color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

// ─── ImageToTensorNCHW ────────────────────────────────────────────────────────

func TestImageToTensorNCHW_OutputLength(t *testing.T) {
	size := 224
	img := solidImage(size, size, color.RGBA{R: 128, G: 64, B: 32, A: 255})
	pixels, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 3 * size * size
	if len(pixels) != want {
		t.Errorf("length: got %d, want %d", len(pixels), want)
	}
}

func TestImageToTensorNCHW_ResizesInput(t *testing.T) {
	// Input is 64×64 but we ask for 224; output should still be 3*224*224.
	img := solidImage(64, 64, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	pixels, err := ImageToTensorNCHW(img, 224)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pixels) != 3*224*224 {
		t.Errorf("length after resize: got %d, want %d", len(pixels), 3*224*224)
	}
}

func TestImageToTensorNCHW_NormalizationRange(t *testing.T) {
	img := solidImage(64, 64, color.RGBA{R: 255, G: 128, B: 0, A: 255})
	pixels, err := ImageToTensorNCHW(img, 64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range pixels {
		if v < 0 || v > 1 {
			t.Errorf("pixel[%d] = %.4f out of [0,1]", i, v)
		}
	}
}

func TestImageToTensorNCHW_ChannelLayout(t *testing.T) {
	// Pure red image: R channel should be ~1, G and B channels should be ~0.
	size := 8
	img := solidImage(size, size, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	pixels, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plane := size * size
	rPlane := pixels[0:plane]
	gPlane := pixels[plane : 2*plane]
	bPlane := pixels[2*plane : 3*plane]

	for i, v := range rPlane {
		if v < 0.99 {
			t.Errorf("R plane[%d] = %.4f, want ~1.0 for pure red image", i, v)
		}
	}
	for i, v := range gPlane {
		if v > 0.01 {
			t.Errorf("G plane[%d] = %.4f, want ~0.0 for pure red image", i, v)
		}
	}
	for i, v := range bPlane {
		if v > 0.01 {
			t.Errorf("B plane[%d] = %.4f, want ~0.0 for pure red image", i, v)
		}
	}
}

func TestImageToTensorNCHW_GreenChannel(t *testing.T) {
	size := 8
	img := solidImage(size, size, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	pixels, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plane := size * size
	rPlane := pixels[0:plane]
	gPlane := pixels[plane : 2*plane]
	bPlane := pixels[2*plane : 3*plane]

	for i, v := range rPlane {
		if v > 0.01 {
			t.Errorf("R plane[%d] = %.4f, want ~0.0 for pure green image", i, v)
		}
	}
	for i, v := range gPlane {
		if v < 0.99 {
			t.Errorf("G plane[%d] = %.4f, want ~1.0 for pure green image", i, v)
		}
	}
	for i, v := range bPlane {
		if v > 0.01 {
			t.Errorf("B plane[%d] = %.4f, want ~0.0 for pure green image", i, v)
		}
	}
}

func TestImageToTensorNCHW_BlackImage(t *testing.T) {
	size := 16
	img := solidImage(size, size, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	pixels, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range pixels {
		if v > 0.01 {
			t.Errorf("pixel[%d] = %.4f, want ~0.0 for black image", i, v)
		}
	}
}

func TestImageToTensorNCHW_WhiteImage(t *testing.T) {
	size := 16
	img := solidImage(size, size, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	pixels, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range pixels {
		if v < 0.99 {
			t.Errorf("pixel[%d] = %.4f, want ~1.0 for white image", i, v)
		}
	}
}

func TestImageToTensorNCHW_DeterministicOutput(t *testing.T) {
	// Same input must always produce the same tensor.
	img := solidImage(32, 32, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	p1, err := ImageToTensorNCHW(img, 32)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	p2, err := ImageToTensorNCHW(img, 32)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	for i := range p1 {
		if p1[i] != p2[i] {
			t.Errorf("pixel[%d] differs: %.6f vs %.6f", i, p1[i], p2[i])
		}
	}
}

func TestImageToTensorNCHW_NoNaNOrInf(t *testing.T) {
	img := solidImage(224, 224, color.RGBA{R: 77, G: 133, B: 210, A: 255})
	pixels, err := ImageToTensorNCHW(img, 224)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range pixels {
		if math.IsNaN(float64(v)) {
			t.Errorf("pixel[%d] is NaN", i)
		}
		if math.IsInf(float64(v), 0) {
			t.Errorf("pixel[%d] is Inf", i)
		}
	}
}

// ─── ImageToTensorNCHW vs ImageToTensorWithSize (NHWC) ───────────────────────

func TestNCHW_vs_NHWC_SamePixelValues(t *testing.T) {
	// Both functions must produce the same set of pixel values — just in
	// different memory layouts.
	size := 16
	img := solidImage(size, size, color.RGBA{R: 80, G: 160, B: 240, A: 255})

	nhwc, err := ImageToTensorWithSize(img, size)
	if err != nil {
		t.Fatalf("NHWC: %v", err)
	}
	nchw, err := ImageToTensorNCHW(img, size)
	if err != nil {
		t.Fatalf("NCHW: %v", err)
	}

	if len(nhwc) != len(nchw) {
		t.Fatalf("length mismatch: NHWC=%d NCHW=%d", len(nhwc), len(nchw))
	}

	// Compute channel sums from both and compare.
	plane := size * size
	var rNHWC, gNHWC, bNHWC float64
	for i := 0; i < plane; i++ {
		rNHWC += float64(nhwc[i*3])
		gNHWC += float64(nhwc[i*3+1])
		bNHWC += float64(nhwc[i*3+2])
	}
	rNCHW := sumSlice(nchw[0:plane])
	gNCHW := sumSlice(nchw[plane : 2*plane])
	bNCHW := sumSlice(nchw[2*plane : 3*plane])

	const tol = 1e-3
	if math.Abs(rNHWC-rNCHW) > tol {
		t.Errorf("R channel sum: NHWC=%.4f NCHW=%.4f", rNHWC, rNCHW)
	}
	if math.Abs(gNHWC-gNCHW) > tol {
		t.Errorf("G channel sum: NHWC=%.4f NCHW=%.4f", gNHWC, gNCHW)
	}
	if math.Abs(bNHWC-bNCHW) > tol {
		t.Errorf("B channel sum: NHWC=%.4f NCHW=%.4f", bNHWC, bNCHW)
	}
}

func sumSlice(s []float32) float64 {
	var total float64
	for _, v := range s {
		total += float64(v)
	}
	return total
}
