package responses

type PreviewManifestResponse struct {
	RecordingID uint     `json:"recordingId" extensions:"!x-nullable"`
	PreviewPath string   `json:"previewPath" extensions:"!x-nullable"`
	FrameCount  uint64   `json:"frameCount" extensions:"!x-nullable"`
	Timestamps  []uint64 `json:"timestamps"`
}
