package preprocessing

import (
	"image"

	"github.com/disintegration/imaging"
	tf "github.com/wamuir/graft/tensorflow"
)

// ImageToTensor converts an image to a TensorFlow tensor with default size (224x224)
// Default size is used for backward compatibility
func ImageToTensor(img image.Image) (*tf.Tensor, error) {
	return ImageToTensorWithSize(img, 224)
}

// ImageToTensorWithSize converts an image to a TensorFlow tensor with specified input size
// The image is resized to size x size using Lanczos interpolation
// Pixel values are normalized to [0, 1] range
// Returns a tensor with shape [1, height, width, 3] suitable for TensorFlow models
func ImageToTensorWithSize(img image.Image, size int) (*tf.Tensor, error) {
	// Resize the image to model input size using high-quality Lanczos filter
	img = imaging.Resize(img, size, size, imaging.Lanczos)

	// Get the image dimensions
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	// Create a slice to hold the pixel data
	// Format: [height][width][3] where 3 is RGB channels
	pixels := make([][][3]float32, height)
	for y := 0; y < height; y++ {
		pixels[y] = make([][3]float32, width)
		for x := 0; x < width; x++ {
			// Get RGBA components and normalize to [0, 1] range
			// RGBA() returns values in [0, 65535] range
			r, g, b, _ := img.At(x, y).RGBA()
			pixels[y][x][0] = float32(r) / 65535.0
			pixels[y][x][1] = float32(g) / 65535.0
			pixels[y][x][2] = float32(b) / 65535.0
		}
	}

	// Create the tensor with batch size of 1
	tensor, err := tf.NewTensor([][][][3]float32{pixels})
	if err != nil {
		return nil, err
	}

	return tensor, nil
}
