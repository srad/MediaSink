package scene

import (
	"fmt"
	"image"
	"math"

	log "github.com/sirupsen/logrus"
	ort "github.com/yalue/onnxruntime_go"

	"github.com/srad/mediasink/internal/analysis/detectors/onnx"
	"github.com/srad/mediasink/internal/analysis/threshold"
	"github.com/srad/mediasink/internal/db"
)

// onnxSceneDetector uses a pre-trained ONNX model to detect scene changes.
type onnxSceneDetector struct {
	session         *ort.DynamicAdvancedSession
	modelConfig     onnx.ModelConfig
	thresholdMethod threshold.ThresholdMethod
}

var _ SceneDetector = (*onnxSceneDetector)(nil)

// NewOnnxSceneDetector creates a new ONNX-based scene detector.
func NewOnnxSceneDetector(modelName string) (SceneDetector, error) {
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

	log.Infof("[OnnxSceneDetector] Loaded model: %s (%s)", modelName, modelConfig.Description())

	return &onnxSceneDetector{
		session:         session,
		modelConfig:     modelConfig,
		thresholdMethod: threshold.NewStatisticalThresholdMethod(2.0),
	}, nil
}

// Close releases the ONNX session resources.
func (d *onnxSceneDetector) Close() error {
	if d.session != nil {
		return d.session.Destroy()
	}
	return nil
}

// Name returns the detector name.
func (d *onnxSceneDetector) Name() string {
	return "onnx"
}

// ExtractFeatures runs inference on a single frame and returns the feature vector.
func (d *onnxSceneDetector) ExtractFeatures(frame image.Image) ([]float32, error) {
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

// DetectScenes detects scenes in a sequence of frames using an adaptive threshold.
// This method is provided for interface compliance; the service uses the streaming
// path (ExtractFeatures + sqlite-vec) which is more efficient.
func (d *onnxSceneDetector) DetectScenes(frames []image.Image, timestamps []float64) ([]db.SceneInfo, error) {
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

	sceneThreshold, err := d.thresholdMethod.Calculate(similarities)
	if err != nil {
		log.Warnf("[ONNX] Failed to calculate adaptive threshold: %v, using fallback", err)
		sceneThreshold = 0.75
	}

	var scenes []db.SceneInfo
	sceneStart := 0.0
	sceneChangeCount := 0

	for i, similarity := range similarities {
		if similarity < sceneThreshold {
			sceneChangeCount++
			scenes = append(scenes, db.SceneInfo{
				StartTime:       sceneStart,
				EndTime:         timestamps[i+1],
				ChangeIntensity: 1.0 - similarity,
			})
			sceneStart = timestamps[i+1]
		}
	}

	if len(frames) > 0 {
		scenes = append(scenes, db.SceneInfo{
			StartTime:       sceneStart,
			EndTime:         timestamps[len(timestamps)-1],
			ChangeIntensity: 0.0,
		})
	}

	totalComparisons := len(similarities)
	triggerRate := float64(sceneChangeCount) / float64(totalComparisons) * 100.0
	log.Infof("[ONNX] Scene detection (%s): %d scenes from %d frames (threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		d.modelConfig.Name(), len(scenes), len(frames), sceneThreshold, d.thresholdMethod.Name(), sceneChangeCount, totalComparisons, triggerRate)

	return scenes, nil
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
