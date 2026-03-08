package onnx

import (
	"gonum.org/v1/gonum/mat"

	"github.com/srad/mediasink/analysis/metrics"
)

// CosineSimilarity calculates cosine similarity between two vectors.
func CosineSimilarity(a, b *mat.VecDense) float64 {
	return metrics.CosineSimilarity(a, b)
}
