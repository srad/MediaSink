package responses

type EstimateEnhancementResponse struct {
	InputFileSize      int64  `json:"inputFileSize" extensions:"!x-nullable"`
	EstimatedFileSize  int64  `json:"estimatedFileSize" extensions:"!x-nullable"`
	EstimatedFileSizeM float64 `json:"estimatedFileSizeMB" extensions:"!x-nullable"`
	CompressionRatio   float64 `json:"compressionRatio" extensions:"!x-nullable"`
}
