package db

import (
	"errors"
	"fmt"
	"strconv"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Setting struct {
	SettingKey   string `json:"settingKey" gorm:"primaryKey;" extensions:"!x-nullable"`
	SettingValue string `json:"settingValue" gorm:"not null;" extensions:"!x-nullable"`
	SettingType  string `json:"-" gorm:"not null;" extensions:"!x-nullable"`
}

const (
	ReqInterval = "req_interval"

	// EmbeddingModelSetting records which model produced the vectors currently
	// stored in frame_vectors.
	EmbeddingModelSetting = "embedding_model"

	// legacyEmbeddingModel is what every install ran before this setting existed.
	legacyEmbeddingModel = "mobilenet_v3_large"
)

// GetEmbeddingModel returns the model that produced the stored frame vectors.
// A missing row means the database predates the setting, so it reports the
// legacy model rather than an error — callers use this to decide whether the
// stored vectors are still comparable with the active model's output.
func GetEmbeddingModel() (string, error) {
	sett := Setting{}
	err := DB.Table("settings").First(&sett, &Setting{SettingKey: EmbeddingModelSetting}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return legacyEmbeddingModel, nil
	}
	if err != nil {
		return "", err
	}
	return sett.SettingValue, nil
}

// SetEmbeddingModel records the model that produced the stored frame vectors.
func SetEmbeddingModel(modelName string) error {
	setting := Setting{SettingKey: EmbeddingModelSetting, SettingValue: modelName, SettingType: "string"}
	return setting.Save()
}

func InitSettings() error {
	if err := DB.FirstOrCreate(
		&Setting{SettingKey: ReqInterval, SettingValue: "15", SettingType: "int"}).Error; err != nil {
		return err
	}

	return nil
}

func GetValue(settingKey string) (interface{}, error) {
	sett := Setting{}

	if err := DB.Table("settings").First(&sett, &Setting{SettingKey: settingKey}).Error; err != nil {
		log.Errorf("[GetValue] Error retreiving setting: %s", err)
		return nil, err
	}

	switch sett.SettingType {
	case "int":
		i, err := strconv.Atoi(sett.SettingValue)
		return i, err
	case "string":
		return sett.SettingValue, nil
	case "bool":
		return sett.SettingValue == "true", nil
	}

	return nil, fmt.Errorf("unknown settings type '%s'", sett.SettingType)
}

func (setting *Setting) Save() error {
	// Use SaveOrCreate to either update existing setting or create new one
	result := DB.Save(setting)
	if result.Error != nil {
		log.Errorf("[Save] Error saving setting: %s", result.Error)
		return result.Error
	}

	// Check if a new record was created
	if result.RowsAffected > 0 {
		log.Infof("[Save] Setting saved: %s = %s", setting.SettingKey, setting.SettingValue)
	}

	return nil
}
