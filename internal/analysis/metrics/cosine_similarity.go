package metrics

import "gonum.org/v1/gonum/mat"

// CosineSimilarity calculates cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical direction
func CosineSimilarity(a, b *mat.VecDense) float64 {
	dot := mat.Dot(a, b)
	na := mat.Norm(a, 2)
	nb := mat.Norm(b, 2)
	return dot / (na * nb)
}
