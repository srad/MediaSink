package requests

type MergeRequest struct {
	RecordingIDs []uint `json:"recordingIds" extensions:"!x-nullable" validate:"required,min=2,dive,gt=0"`
	ReEncode     bool   `json:"reEncode" extensions:"!x-nullable"`
}
