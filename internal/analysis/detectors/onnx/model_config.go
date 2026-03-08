package onnx

import (
	"fmt"
	"image"
	"os"
	"sync"

	ort "github.com/yalue/onnxruntime_go"

	"github.com/srad/mediasink/internal/analysis/preprocessing"
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

// MobileNetV3Config is the configuration for the MobileNet V3 Large model.
type MobileNetV3Config struct{}

func (m *MobileNetV3Config) Name() string      { return "mobilenet_v3_large" }
func (m *MobileNetV3Config) InputSize() int    { return 224 }
func (m *MobileNetV3Config) InputName() string { return "input" }
func (m *MobileNetV3Config) OutputName() string { return "output" }
func (m *MobileNetV3Config) Description() string {
	return "MobileNet V3 Large - Lightweight feature extractor, 224x224 input"
}
func (m *MobileNetV3Config) PreprocessFrame(frame image.Image) ([]float32, error) {
	return preprocessing.ImageToTensorNCHW(frame, m.InputSize())
}

// GetModelConfig returns the configuration for the given model name.
func GetModelConfig(modelName string) (ModelConfig, error) {
	switch modelName {
	case "mobilenet_v3_large":
		return &MobileNetV3Config{}, nil
	default:
		return nil, fmt.Errorf("unknown model: %s", modelName)
	}
}
