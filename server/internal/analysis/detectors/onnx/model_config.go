package onnx

import (
	"fmt"
	"image"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/srad/mediasink/server/internal/analysis/preprocessing"
)

var (
	initOnce sync.Once
	initErr  error
)

// EnsureInitialized initializes the ONNX runtime environment once.
// The shared library path can be overridden with the ONNXRUNTIME_LIB env var.
func EnsureInitialized() error {
	initOnce.Do(func() {
		if ort.IsInitialized() {
			return
		}
		if libPath := os.Getenv("ONNXRUNTIME_LIB"); libPath != "" {
			ort.SetSharedLibraryPath(libPath)
		}
		initErr = ort.InitializeEnvironment()
	})
	return initErr
}

// ModelConfig defines the configuration for an ONNX model.
type ModelConfig interface {
	// Name returns the model identifier (matches the .onnx filename without extension).
	Name() string

	// InputSize returns the expected square input resolution.
	InputSize() int

	// InputName returns the ONNX model input node name.
	InputName() string

	// OutputName returns the ONNX model output node name.
	OutputName() string

	// PreprocessFrame resizes and normalizes the image, returning a flat []float32
	// of length InputSize*InputSize*3 in row-major NHWC order (batch omitted).
	PreprocessFrame(frame image.Image) ([]float32, error)

	// Description returns a human-readable description of the model.
	Description() string
}

// DefaultModelName is the embedding model the server currently runs on. Frame
// vectors are only comparable within one model, so changing this invalidates
// every stored embedding — see the frame_vectors reset in internal/db/db.go.
const DefaultModelName = "mobilenetv4_conv_large"

// MobileNetV4LargeConfig is the configuration for the MobileNet V4 Large model.
// The exported graph applies ImageNet normalization internally, so it takes the
// same [0,1] NCHW input as the V3 model — see scripts/export_mobilenetv4_onnx.py.
type MobileNetV4LargeConfig struct{}

func (m *MobileNetV4LargeConfig) Name() string       { return DefaultModelName }
func (m *MobileNetV4LargeConfig) InputSize() int     { return 256 }
func (m *MobileNetV4LargeConfig) InputName() string  { return "input" }
func (m *MobileNetV4LargeConfig) OutputName() string { return "output" }
func (m *MobileNetV4LargeConfig) Description() string {
	return "MobileNet V4 Large - Feature extractor, 256x256 input"
}
func (m *MobileNetV4LargeConfig) PreprocessFrame(frame image.Image) ([]float32, error) {
	return preprocessing.ImageToTensorNCHW(frame, m.InputSize())
}

// GetModelConfig returns the configuration for the given model name.
func GetModelConfig(modelName string) (ModelConfig, error) {
	switch modelName {
	case DefaultModelName:
		return &MobileNetV4LargeConfig{}, nil
	default:
		return nil, fmt.Errorf("unknown model: %s", modelName)
	}
}
