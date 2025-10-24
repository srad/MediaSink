package requests

type EstimateEnhancementRequest struct {
	TargetResolution string  `json:"targetResolution" extensions:"!x-nullable" validate:"required,oneof=720p 1080p 1440p 4k"`
	DenoiseStrength  float64 `json:"denoiseStrength" extensions:"!x-nullable" validate:"required,min=1.0,max=10.0"`
	SharpenStrength  float64 `json:"sharpenStrength" extensions:"!x-nullable" validate:"required,min=0.0,max=2.0"`
	ApplyNormalize   bool    `json:"applyNormalize" extensions:"!x-nullable"`
	EncodingPreset   string  `json:"encodingPreset" extensions:"!x-nullable" validate:"required,oneof=veryfast faster fast medium slow slower veryslow"`
	CRF              *uint   `json:"crf" validate:"omitempty,min=15,max=28"`
}
