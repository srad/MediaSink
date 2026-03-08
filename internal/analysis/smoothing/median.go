package smoothing

import "sort"

// medianSmoothing implements median filter smoothing (best edge preservation)
type medianSmoothing struct{}

var _ SmoothingMethod = (*medianSmoothing)(nil)

func (m *medianSmoothing) Name() string {
	return "median"
}

func (m *medianSmoothing) Smooth(data []float64, windowSize int) []float64 {
	if windowSize <= 1 || len(data) <= windowSize {
		return data
	}

	smoothed := make([]float64, len(data))
	half := windowSize / 2

	for i := 0; i < len(data); i++ {
		start := i - half
		if start < 0 {
			start = 0
		}
		end := i + half + 1
		if end > len(data) {
			end = len(data)
		}

		// Extract window values
		window := make([]float64, end-start)
		copy(window, data[start:end])

		// Sort to find median
		sort.Float64s(window)

		// Get median value
		mid := len(window) / 2
		if len(window)%2 == 0 {
			// Even number of elements: average of middle two
			smoothed[i] = (window[mid-1] + window[mid]) / 2.0
		} else {
			// Odd number of elements: middle element
			smoothed[i] = window[mid]
		}
	}

	return smoothed
}
