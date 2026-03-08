package preprocessing

import (
	"image"

	"github.com/disintegration/imaging"
)

// ImageToTensor converts an image to a flat float32 slice with default size (224x224)
func ImageToTensor(img image.Image) ([]float32, error) {
	return ImageToTensorWithSize(img, 224)
}

// ImageToTensorWithSize converts an image to a flat float32 slice of length H*W*3.
// The image is resized to size x size using Lanczos interpolation.
// Pixel values are normalized to [0, 1] range.
// Layout: row-major NHWC, batch dimension omitted (caller wraps in shape [1,H,W,3]).
func ImageToTensorWithSize(img image.Image, size int) ([]float32, error) {
	img = imaging.Resize(img, size, size, imaging.Lanczos)

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	pixels := make([]float32, height*width*3)
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// RGBA() returns values in [0, 65535]; normalize to [0, 1]
			r, g, b, _ := img.At(x, y).RGBA()
			pixels[idx] = float32(r) / 65535.0
			pixels[idx+1] = float32(g) / 65535.0
			pixels[idx+2] = float32(b) / 65535.0
			idx += 3
		}
	}

	return pixels, nil
}

// ImageToTensorNCHW converts an image to a flat float32 slice in NCHW layout
// (channel-first): all R values, then all G values, then all B values.
// The image is resized to size x size. Pixel values are normalized to [0, 1].
// Caller wraps in shape [1, 3, size, size].
func ImageToTensorNCHW(img image.Image, size int) ([]float32, error) {
	img = imaging.Resize(img, size, size, imaging.Lanczos)

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y
	planeSize := height * width

	pixels := make([]float32, 3*planeSize)
	rBase := 0
	gBase := planeSize
	bBase := 2 * planeSize

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*width + x
			r, g, b, _ := img.At(x, y).RGBA()
			pixels[rBase+i] = float32(r) / 65535.0
			pixels[gBase+i] = float32(g) / 65535.0
			pixels[bBase+i] = float32(b) / 65535.0
		}
	}

	return pixels, nil
}
