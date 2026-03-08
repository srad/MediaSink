package highlight

import (
	"fmt"
	"image"
	"math"

	log "github.com/sirupsen/logrus"
	ort "github.com/yalue/onnxruntime_go"

	"github.com/srad/mediasink/analysis/detectors/onnx"
	"github.com/srad/mediasink/analysis/threshold"
	"github.com/srad/mediasink/database"
)

// onnxHighlightDetector uses a pre-trained ONNX model to detect highlights.
type onnxHighlightDetector struct {
	session         *ort.DynamicAdvancedSession
	modelConfig     onnx.ModelConfig
	thresholdMethod threshold.ThresholdMethod
}

var _ HighlightDetector = (*onnxHighlightDetector)(nil)

// NewOnnxHighlightDetector creates a new ONNX-based highlight detector.
func NewOnnxHighlightDetector(modelName string) (HighlightDetector, error) {
	modelConfig, err := onnx.GetModelConfig(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get model config: %w", err)
	}

	modelPath, err := onnx.GetModelPath(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find model path: %w", err)
	}

	if err := onnx.EnsureInitialized(); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX runtime: %w", err)
	}

	session, err := ort.NewDynamicAdvancedSession(modelPath,
		[]string{modelConfig.InputName()},
		[]string{modelConfig.OutputName()},
		nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create ONNX session for %s: %w", modelName, err)
	}

	log.Infof("[OnnxHighlightDetector] Loaded model: %s (%s)", modelName, modelConfig.Description())

	return &onnxHighlightDetector{
		session:         session,
		modelConfig:     modelConfig,
		thresholdMethod: threshold.NewStatisticalThresholdMethod(3.0),
	}, nil
}

// Close releases the ONNX session resources.
func (d *onnxHighlightDetector) Close() error {
	if d.session != nil {
		return d.session.Destroy()
	}
	return nil
}

// Name returns the detector name.
func (d *onnxHighlightDetector) Name() string {
	return "onnx"
}

// ExtractFeatures runs inference on a single frame and returns the feature vector.
func (d *onnxHighlightDetector) ExtractFeatures(frame image.Image) ([]float32, error) {
	size := d.modelConfig.InputSize()

	flatPixels, err := d.modelConfig.PreprocessFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("preprocessing failed: %w", err)
	}

	inputTensor, err := ort.NewTensor(ort.NewShape(1, 3, int64(size), int64(size)), flatPixels)
	if err != nil {
		return nil, fmt.Errorf("failed to create input tensor: %w", err)
	}
	defer inputTensor.Destroy()

	outputs := []ort.Value{nil}
	if err := d.session.Run([]ort.Value{inputTensor}, outputs); err != nil {
		return nil, fmt.Errorf("inference failed: %w", err)
	}
	defer outputs[0].Destroy()

	outputTensor, ok := outputs[0].(*ort.Tensor[float32])
	if !ok {
		return nil, fmt.Errorf("unexpected output tensor type")
	}

	return outputTensor.GetData(), nil
}

// DetectHighlights detects highlights in a sequence of frames using an adaptive threshold.
// This method is provided for interface compliance; the service uses the streaming
// path (ExtractFeatures + sqlite-vec) which is more efficient.
func (d *onnxHighlightDetector) DetectHighlights(frames []image.Image, timestamps []float64) ([]database.HighlightInfo, error) {
	if len(frames) < 2 {
		return nil, nil
	}

	var vectors [][]float32
	for _, frame := range frames {
		vec, err := d.ExtractFeatures(frame)
		if err != nil {
			return nil, err
		}
		vectors = append(vectors, vec)
	}

	var similarities []float64
	for i := 1; i < len(vectors); i++ {
		similarities = append(similarities, cosineSim(vectors[i-1], vectors[i]))
	}

	highlightThreshold, err := d.thresholdMethod.Calculate(similarities)
	if err != nil {
		log.Warnf("[ONNX] Failed to calculate adaptive threshold: %v, using fallback", err)
		highlightThreshold = 0.5
	}

	var highlights []database.HighlightInfo
	highlightCount := 0

	for i, similarity := range similarities {
		if similarity < highlightThreshold {
			highlightCount++
			highlights = append(highlights, database.HighlightInfo{
				Timestamp: timestamps[i+1],
				Intensity: 1.0 - similarity,
				Type:      "motion",
			})
		}
	}

	triggerRate := float64(highlightCount) / float64(len(similarities)) * 100.0
	log.Infof("[ONNX] Highlight detection (%s): %d highlights from %d frames (threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		d.modelConfig.Name(), len(highlights), len(frames), highlightThreshold, d.thresholdMethod.Name(), highlightCount, len(similarities), triggerRate)

	return highlights, nil
}

// cosineSim computes cosine similarity between two float32 vectors.
func cosineSim(a, b []float32) float64 {
	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
