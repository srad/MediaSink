package smoothing

import (
	"math"
	"testing"
)

func allMethods() []SmoothingMethod {
	return []SmoothingMethod{
		&medianSmoothing{},
		&movingAverageSmoothing{},
		&gaussianSmoothing{},
		&noSmoothing{},
	}
}

// The Gaussian kernel used to be allocated with length windowSize while the
// loop filled 2*half+1 entries, so every even window panicked.
func TestGaussianEvenWindowSizes(t *testing.T) {
	data := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	for _, windowSize := range []int{2, 3, 4, 5, 6, 7} {
		smoothed := (&gaussianSmoothing{}).Smooth(data, windowSize)
		if len(smoothed) != len(data) {
			t.Fatalf("windowSize %d: got length %d, want %d", windowSize, len(smoothed), len(data))
		}
	}
}

// A correctly normalized kernel leaves a constant signal untouched.
func TestGaussianPreservesConstantSignal(t *testing.T) {
	data := make([]float64, 20)
	for i := range data {
		data[i] = 0.75
	}

	for _, windowSize := range []int{2, 3, 4, 5} {
		for i, v := range (&gaussianSmoothing{}).Smooth(data, windowSize) {
			if math.Abs(v-0.75) > 1e-9 {
				t.Fatalf("windowSize %d index %d: got %v, want 0.75", windowSize, i, v)
			}
		}
	}
}

// Callers must always own the returned slice, including on the short-circuit
// path where the input used to be handed straight back.
func TestSmoothNeverAliasesInput(t *testing.T) {
	for _, method := range allMethods() {
		data := []float64{1, 2, 3}
		original := data[0]

		smoothed := method.Smooth(data, 5) // len(data) <= windowSize
		smoothed[0] = 99

		if data[0] != original {
			t.Errorf("%s: mutating the result changed the input", method.Name())
		}
	}
}

func TestSmoothPreservesLength(t *testing.T) {
	data := []float64{0.9, 0.2, 0.8, 0.15, 0.95, 0.3, 0.85, 0.1}

	for _, method := range allMethods() {
		for _, windowSize := range []int{1, 2, 3, 4, 5} {
			if got := len(method.Smooth(data, windowSize)); got != len(data) {
				t.Errorf("%s windowSize %d: got length %d, want %d",
					method.Name(), windowSize, got, len(data))
			}
		}
	}
}
