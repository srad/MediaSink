package smoothing

// movingAverageSmoothing implements simple moving average smoothing
type movingAverageSmoothing struct{}

var _ SmoothingMethod = (*movingAverageSmoothing)(nil)

func (m *movingAverageSmoothing) Name() string {
	return "moving-average"
}

func (m *movingAverageSmoothing) Smooth(data []float64, windowSize int) []float64 {
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

		sum := 0.0
		for j := start; j < end; j++ {
			sum += data[j]
		}
		smoothed[i] = sum / float64(end-start)
	}

	return smoothed
}
