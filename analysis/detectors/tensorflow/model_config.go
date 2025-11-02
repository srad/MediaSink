package tensorflow

import (
	"fmt"
	"image"

	tf "github.com/wamuir/graft/tensorflow"
	"github.com/srad/mediasink/analysis/preprocessing"
)

// ModelConfig defines the configuration for a TensorFlow model
// Each model can have different input sizes, preprocessing steps, normalization, etc.
type ModelConfig interface {
	// Name returns the model identifier
	Name() string

	// InputSize returns the expected input resolution (assumed square: InputSize x InputSize)
	InputSize() int

	// PreprocessFrame converts an image to a TensorFlow tensor with model-specific preprocessing
	PreprocessFrame(frame image.Image) (*tf.Tensor, error)

	// Description returns a human-readable description of the model
	Description() string
}

// MobileNetV2Config configuration for MobileNet V2 model
type MobileNetV2Config struct{}

func (m *MobileNetV2Config) Name() string {
	return "mobilenet_v2"
}

func (m *MobileNetV2Config) InputSize() int {
	return 224
}

func (m *MobileNetV2Config) Description() string {
	return "MobileNet V2 - Lightweight feature extractor, 224x224 input"
}

func (m *MobileNetV2Config) PreprocessFrame(frame image.Image) (*tf.Tensor, error) {
	return preprocessing.ImageToTensorWithSize(frame, m.InputSize())
}

// MobileNetV3Config configuration for MobileNet V3 Large model
type MobileNetV3Config struct{}

func (m *MobileNetV3Config) Name() string {
	return "mobilenet_v3_large"
}

func (m *MobileNetV3Config) InputSize() int {
	return 224
}

func (m *MobileNetV3Config) Description() string {
	return "MobileNet V3 Large - Improved lightweight feature extractor, 224x224 input"
}

func (m *MobileNetV3Config) PreprocessFrame(frame image.Image) (*tf.Tensor, error) {
	return preprocessing.ImageToTensorWithSize(frame, m.InputSize())
}

// MobileViTConfig configuration for MobileViT-XXS model
type MobileViTConfig struct{}

func (m *MobileViTConfig) Name() string {
	return "mobilevit"
}

func (m *MobileViTConfig) InputSize() int {
	return 256
}

func (m *MobileViTConfig) Description() string {
	return "MobileViT-XXS - Vision Transformer model, 256x256 input"
}

func (m *MobileViTConfig) PreprocessFrame(frame image.Image) (*tf.Tensor, error) {
	return preprocessing.ImageToTensorWithSize(frame, m.InputSize())
}

// GetModelConfig returns the configuration for a given model name
// Returns error if the model is not recognized
func GetModelConfig(modelName string) (ModelConfig, error) {
	switch modelName {
	case "mobilenet_v2":
		return &MobileNetV2Config{}, nil
	case "mobilenet_v3_large":
		return &MobileNetV3Config{}, nil
	case "mobilevit":
		return &MobileViTConfig{}, nil
	default:
		return nil, fmt.Errorf("unknown model: %s", modelName)
	}
}

// ListAvailableModels returns all available model configurations
func ListAvailableModels() []ModelConfig {
	return []ModelConfig{
		&MobileNetV2Config{},
		&MobileNetV3Config{},
		&MobileViTConfig{},
	}
}
