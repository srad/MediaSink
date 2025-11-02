package database

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

type SceneInfo struct {
	StartTime       float64 `json:"startTime"`
	EndTime         float64 `json:"endTime"`
	ChangeIntensity float64 `json:"changeIntensity"` // 0-1, higher = more change
}

type HighlightInfo struct {
	Timestamp float64 `json:"timestamp"`
	Intensity float64 `json:"intensity"` // 0-1, higher = more activity
	Type      string  `json:"type"`      // "motion", "sceneChange", "transition"
}

type VideoAnalysisResult struct {
	AnalysisID  uint            `json:"analysisId" gorm:"autoIncrement;primaryKey;column:analysis_id" extensions:"!x-nullable"`
	RecordingID RecordingID     `json:"recordingId" gorm:"not null;unique;index;column:recording_id" extensions:"!x-nullable"`
	Recording   Recording       `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;foreignKey:recording_id;references:recording_id"`
	Status      AnalysisStatus  `json:"status" gorm:"not null;default:pending" extensions:"!x-nullable"`
	ScenesJSON  json.RawMessage `json:"scenesRaw" gorm:"type:json" extensions:"!x-nullable"`
	HighlightsJSON json.RawMessage `json:"highlightsRaw" gorm:"type:json" extensions:"!x-nullable"`
	CreatedAt   time.Time       `json:"createdAt" gorm:"not null;default:current_timestamp" extensions:"!x-nullable"`
	UpdatedAt   time.Time       `json:"updatedAt" gorm:"not null;default:current_timestamp" extensions:"!x-nullable"`
	Error       string          `json:"error" gorm:"default:null"`
}

type AnalysisStatus string

const (
	AnalysisPending    AnalysisStatus = "pending"
	AnalysisProcessing AnalysisStatus = "processing"
	AnalysisCompleted  AnalysisStatus = "completed"
	AnalysisError      AnalysisStatus = "error"
)

func (v *VideoAnalysisResult) TableName() string {
	return "video_analyses"
}

func (v *VideoAnalysisResult) GetScenes() ([]SceneInfo, error) {
	if len(v.ScenesJSON) == 0 {
		return []SceneInfo{}, nil
	}
	var scenes []SceneInfo
	if err := json.Unmarshal(v.ScenesJSON, &scenes); err != nil {
		return nil, err
	}
	return scenes, nil
}

func (v *VideoAnalysisResult) GetHighlights() ([]HighlightInfo, error) {
	if len(v.HighlightsJSON) == 0 {
		return []HighlightInfo{}, nil
	}
	var highlights []HighlightInfo
	if err := json.Unmarshal(v.HighlightsJSON, &highlights); err != nil {
		return nil, err
	}
	return highlights, nil
}

func (v *VideoAnalysisResult) SetScenes(scenes []SceneInfo) error {
	data, err := json.Marshal(scenes)
	if err != nil {
		return err
	}
	v.ScenesJSON = data
	return nil
}

func (v *VideoAnalysisResult) SetHighlights(highlights []HighlightInfo) error {
	data, err := json.Marshal(highlights)
	if err != nil {
		return err
	}
	v.HighlightsJSON = data
	return nil
}

func CreateOrUpdateAnalysis(recordingID RecordingID) (*VideoAnalysisResult, error) {
	analysis := &VideoAnalysisResult{
		RecordingID: recordingID,
		Status:      AnalysisPending,
	}

	result := DB.Where("recording_id = ?", recordingID).
		Assign(analysis).
		FirstOrCreate(analysis)

	if result.Error != nil {
		return nil, result.Error
	}

	return analysis, nil
}

func GetAnalysisByRecordingID(recordingID RecordingID) (*VideoAnalysisResult, error) {
	var analysis *VideoAnalysisResult
	err := DB.Where("recording_id = ?", recordingID).First(&analysis).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return analysis, nil
}

func (v *VideoAnalysisResult) UpdateStatus(status AnalysisStatus) error {
	return DB.Model(v).Update("status", status).Error
}

func (v *VideoAnalysisResult) UpdateError(errMsg string) error {
	return DB.Model(v).Updates(map[string]interface{}{
		"status": AnalysisError,
		"error":  errMsg,
	}).Error
}

func (v *VideoAnalysisResult) SaveResults() error {
	return DB.Model(v).Updates(map[string]interface{}{
		"status":          AnalysisCompleted,
		"scenes_json":     v.ScenesJSON,
		"highlights_json": v.HighlightsJSON,
		"error":           nil,
	}).Error
}

// DeleteAnalysisByRecordingID deletes any existing analysis for a recording
func DeleteAnalysisByRecordingID(recordingID RecordingID) error {
	return DB.Where("recording_id = ?", recordingID).Delete(&VideoAnalysisResult{}).Error
}
