package highlight
import (
	"fmt"
	"image"
	log "github.com/sirupsen/logrus"
	tf "github.com/wamuir/graft/tensorflow"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/analysis/detectors/tensorflow"
	"github.com/srad/mediasink/analysis/threshold"
	"github.com/srad/mediasink/analysis/metrics"
	"gonum.org/v1/gonum/mat"
)
const (
	tfHighlightThreshold = 0.8
)
// tensorFlowHighlightDetector uses a pre-trained TensorFlow model to detect highlights.
type tensorFlowHighlightDetector struct {
	model            *tf.SavedModel
	session          *tf.Session
	inputOp          tf.Output
	outputOp         tf.Output
	modelConfig      tensorflow.ModelConfig // Use config interface for model-specific handling
	thresholdMethod  threshold.ThresholdMethod
}

var _ HighlightDetector = (*tensorFlowHighlightDetector)(nil)

// NewTensorFlowHighlightDetector creates a new TensorFlowHighlightDetector.
func NewTensorFlowHighlightDetector(modelName string) (HighlightDetector, error) {
	// Get model configuration
	modelConfig, err := tensorflow.GetModelConfig(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to get model config: %w", err)
	}
	modelPath, err := tensorflow.GetModelPath(modelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find model path: %w", err)
	}
	// Load SavedModel (not ONNX - official TF Go doesn't support ONNX)
	loadedModel, err := tf.LoadSavedModel(modelPath, []string{"serve"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load SavedModel: %w", err)
	}
	// Get the signature to find input/output operations
	// Most TensorFlow Hub models use "serving_default" signature
	sigDef, ok := loadedModel.Signatures["serving_default"]
	if !ok {
		return nil, fmt.Errorf("model does not have 'serving_default' signature")
	}
	// Get first input and output tensor info from signature
	var inputInfo, outputInfo tf.TensorInfo
	var inputName, outputName string
	for name, input := range sigDef.Inputs {
		inputInfo = input
		inputName = name
		break
	}
	for name, output := range sigDef.Outputs {
		outputInfo = output
		outputName = name
		break
	}
	if inputInfo.Name == "" || outputInfo.Name == "" {
		return nil, fmt.Errorf("signature input/output names are empty (input key: %s, output key: %s)", inputName, outputName)
	}
	// TensorInfo.Name contains the full tensor name like "operation:0"
	// Parse to get operation name and output index
	inputOpName, inputIdx, err := tensorflow.ParseTensorName(inputInfo.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input tensor name: %w", err)
	}
	outputOpName, outputIdx, err := tensorflow.ParseTensorName(outputInfo.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to parse output tensor name: %w", err)
	}
	// Get operations from graph and create outputs
	inputOp := loadedModel.Graph.Operation(inputOpName).Output(inputIdx)
	outputOp := loadedModel.Graph.Operation(outputOpName).Output(outputIdx)
	if inputOp.Op == nil || outputOp.Op == nil {
		return nil, fmt.Errorf("failed to get operations (input: %s, output: %s)", inputInfo.Name, outputInfo.Name)
	}
	log.Infof("[TensorFlowHighlightDetector] Loaded model: %s (%s)", modelName, modelConfig.Description())
	return &tensorFlowHighlightDetector{
		model:           loadedModel,
		session:         loadedModel.Session,
		inputOp:         inputOp,
		outputOp:        outputOp,
		modelConfig:     modelConfig,
		thresholdMethod: threshold.NewStatisticalThresholdMethod(3.0), // k=3.0 for highlight detection
	}, nil
}
// Close releases resources
func (d *tensorFlowHighlightDetector) Close() error {
	if d.session != nil {
		return d.session.Close()
	}
	return nil
}
// Name returns the name of the detector.
func (d *tensorFlowHighlightDetector) Name() string {
	return "tensorflow"
}
// DetectHighlights detects highlights in a sequence of frames with adaptive threshold.
func (d *tensorFlowHighlightDetector) DetectHighlights(frames []image.Image, timestamps []float64) ([]database.HighlightInfo, error) {
	if len(frames) < 2 {
		return nil, nil
	}
	// Extract all feature vectors
	var vectors []*mat.VecDense
	for _, frame := range frames {
		// Use model-specific preprocessing
		tensor, err := d.modelConfig.PreprocessFrame(frame)
		if err != nil {
			return nil, err
		}
		// Run inference using the session
		outputs, err := d.session.Run(
			map[tf.Output]*tf.Tensor{
				d.inputOp: tensor,
			},
			[]tf.Output{
				d.outputOp,
			},
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("inference failed: %w", err)
		}
		// Extract features from output tensor
		features := outputs[0].Value().([][]float32)[0]
		vector := mat.NewVecDense(len(features), nil)
		for j, f := range features {
			vector.SetVec(j, float64(f))
		}
		vectors = append(vectors, vector)
	}
	// Calculate frame-to-frame similarities
	var similarities []float64
	for i := 1; i < len(vectors); i++ {
		similarity := metrics.CosineSimilarity(vectors[i-1], vectors[i])
		similarities = append(similarities, similarity)
	}
	// Calculate adaptive threshold
	threshold, err := d.thresholdMethod.Calculate(similarities)
	if err != nil {
		log.Warnf("[TensorFlow] Failed to calculate adaptive threshold: %v, using fallback", err)
		threshold = 0.5 // Fallback threshold
	}
	// Detect highlights using calculated threshold
	var highlights []database.HighlightInfo
	highlightCount := 0
	for i := 0; i < len(similarities); i++ {
		similarity := similarities[i]
		// Record highlight if similarity is below threshold (motion detected)
		if similarity < threshold {
			highlightCount++
			highlights = append(highlights, database.HighlightInfo{
				Timestamp: timestamps[i+1],
				Intensity: 1.0 - similarity,
				Type:      "motion",
			})
		}
	}
	triggerRate := float64(highlightCount) / float64(len(similarities)) * 100.0
	log.Infof("[TensorFlow] Highlight detection (%s): %d highlights from %d frames (adaptive threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		d.modelConfig.Name(), len(highlights), len(frames), threshold, d.thresholdMethod.Name(), highlightCount, len(similarities), triggerRate)
	return highlights, nil
}
