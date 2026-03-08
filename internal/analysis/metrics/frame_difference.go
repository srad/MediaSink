package metrics

import (
	"image"
	"math"
)

// Constants for frame difference calculation
const (
	rgbChannels     = 3.0
	maxColorValue   = 255.0
)

// CalculateFrameDifference computes the normalized pixel-level difference between two frames
// Uses mean squared error (MSE) on RGB channels
// Returns value between 0 and 1, where 0 means identical and 1 means completely different
func CalculateFrameDifference(img1, img2 image.Image) float64 {
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	// Use average dimensions
	width := (bounds1.Dx() + bounds2.Dx()) / 2
	height := (bounds1.Dy() + bounds2.Dy()) / 2

	var sumDiff float64
	pixelCount := 0

	// Sample pixels from both images
	for y := 0; y < height && y < bounds1.Dy() && y < bounds2.Dy(); y++ {
		for x := 0; x < width && x < bounds1.Dx() && x < bounds2.Dx(); x++ {
			r1, g1, b1, _ := img1.At(bounds1.Min.X+x, bounds1.Min.Y+y).RGBA()
			r2, g2, b2, _ := img2.At(bounds2.Min.X+x, bounds2.Min.Y+y).RGBA()

			// Calculate RGB difference
			rDiff := float64(r1>>8) - float64(r2>>8)
			gDiff := float64(g1>>8) - float64(g2>>8)
			bDiff := float64(b1>>8) - float64(b2>>8)

			diff := (rDiff*rDiff + gDiff*gDiff + bDiff*bDiff) / (rgbChannels * maxColorValue * maxColorValue)
			sumDiff += diff
			pixelCount++
		}
	}

	if pixelCount == 0 {
		return 0.0
	}

	// Return normalized average difference
	avgDiff := sumDiff / float64(pixelCount)
	return math.Min(avgDiff, 1.0)
}
