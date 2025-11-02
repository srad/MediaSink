package metrics

import "image"

// Constants for SSIM calculation (stability constants)
const (
	ssimC1 = 6.5025  // (0.01 * 255)^2
	ssimC2 = 58.5225 // (0.03 * 255)^2
)

// CalculateSSIM computes the Structural Similarity Index (SSIM) between two images
// SSIM measures perceived quality degradation caused by transmission in Video compression
// Returns value between 0 and 1, where 1 indicates identical images
func CalculateSSIM(img1, img2 image.Image) float64 {
	// Normalize images to same bounds if needed
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()

	// Use average dimensions
	width := (bounds1.Dx() + bounds2.Dx()) / 2
	height := (bounds1.Dy() + bounds2.Dy()) / 2

	// Calculate mean luminance for each image
	var sum1, sum2, sumSq1, sumSq2, sumProd float64
	pixelCount := 0

	// Sample pixels from both images
	for y := 0; y < height && y < bounds1.Dy() && y < bounds2.Dy(); y++ {
		for x := 0; x < width && x < bounds1.Dx() && x < bounds2.Dx(); x++ {
			r1, g1, b1, _ := img1.At(bounds1.Min.X+x, bounds1.Min.Y+y).RGBA()
			r2, g2, b2, _ := img2.At(bounds2.Min.X+x, bounds2.Min.Y+y).RGBA()

			// Convert to luminance (grayscale) using standard BT.601 weights
			lum1 := float64(r1>>8)*0.299 + float64(g1>>8)*0.587 + float64(b1>>8)*0.114
			lum2 := float64(r2>>8)*0.299 + float64(g2>>8)*0.587 + float64(b2>>8)*0.114

			sum1 += lum1
			sum2 += lum2
			sumSq1 += lum1 * lum1
			sumSq2 += lum2 * lum2
			sumProd += lum1 * lum2
			pixelCount++
		}
	}

	if pixelCount == 0 {
		return 1.0
	}

	float64PixelCount := float64(pixelCount)
	mean1 := sum1 / float64PixelCount
	mean2 := sum2 / float64PixelCount
	var1 := (sumSq1 / float64PixelCount) - mean1*mean1
	var2 := (sumSq2 / float64PixelCount) - mean2*mean2
	cov := (sumProd / float64PixelCount) - mean1*mean2

	// SSIM formula: (2*mean1*mean2 + C1) * (2*cov + C2) / ((mean1^2 + mean2^2 + C1) * (var1 + var2 + C2))
	numerator := (2*mean1*mean2 + ssimC1) * (2*cov + ssimC2)
	denominator := (mean1*mean1 + mean2*mean2 + ssimC1) * (var1 + var2 + ssimC2)

	if denominator == 0 {
		return 1.0
	}

	ssim := numerator / denominator
	if ssim < 0 {
		return 0.0
	}
	if ssim > 1.0 {
		return 1.0
	}

	return ssim
}
