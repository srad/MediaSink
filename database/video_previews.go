package database

import (
	"time"
)

type VideoPreviewID uint

type VideoPreview struct {
	VideoPreviewID VideoPreviewID `json:"videoPreviewId" gorm:"autoIncrement;primaryKey;column:video_preview_id" extensions:"!x-nullable" validate:"gte=0"`

	Recording   Recording   `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:recording_id;references:recording_id" validate:"-"`
	RecordingID RecordingID `json:"recordingId" gorm:"not null;default:null;column:recording_id" extensions:"!x-nullable" validate:"gte=0"`

	FrameCount    uint64 `json:"frameCount" gorm:"not null;default:0" extensions:"!x-nullable"`
	FrameInterval uint64 `json:"frameInterval" gorm:"not null;default:0" extensions:"!x-nullable"`
	PreviewPath   string `json:"previewPath" gorm:"not null;default:null" extensions:"!x-nullable" validate:"required,filepath"`
	CreatedAt     time.Time `json:"createdAt" gorm:"not null;default:current_timestamp" extensions:"!x-nullable"`
	UpdatedAt     time.Time `json:"updatedAt" gorm:"not null;default:current_timestamp" extensions:"!x-nullable"`
}

func (VideoPreview) TableName() string {
	return "video_previews"
}

// CreateVideoPreview creates a new video preview record
func (vp *VideoPreview) CreateVideoPreview() error {
	return DB.Create(vp).Error
}

// FindVideoPreviewByRecordingID retrieves preview metadata for a recording
func FindVideoPreviewByRecordingID(recordingID RecordingID) (*VideoPreview, error) {
	var preview VideoPreview
	if err := DB.Where("recording_id = ?", recordingID).First(&preview).Error; err != nil {
		return nil, err
	}
	return &preview, nil
}

// UpdateVideoPreview updates an existing video preview record
func (vp *VideoPreview) UpdateVideoPreview() error {
	return DB.Model(vp).Updates(vp).Error
}

// DeleteVideoPreviewByRecordingID deletes preview metadata for a recording
func DeleteVideoPreviewByRecordingID(recordingID RecordingID) error {
	return DB.Where("recording_id = ?", recordingID).Delete(&VideoPreview{}).Error
}
