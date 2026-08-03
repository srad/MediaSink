package db

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Setting struct {
	SettingKey string `json:"settingKey" gorm:"primaryKey;" extensions:"!x-nullable"`
	// SettingValue is the only field read back today. SettingType is still written on
	// every save but has had no reader since the generic type-switching getter was
	// deleted in phase 2b.2 (it had no callers); the column is kept because dropping a
	// persisted one is a migration, not a refactor.
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

// SettingRepo is the settings aggregate. Obtained from Store.Settings(), so it always
// carries the handle its store was built with. That matters more here than elsewhere:
// Migrate reads and seeds settings while running against a database that is not
// necessarily the one the package global points at.
type SettingRepo struct {
	gorm *gorm.DB
}

// EmbeddingModel returns the model that produced the stored frame vectors.
// A missing row means the database predates the setting, so it reports the
// legacy model rather than an error — callers use this to decide whether the
// stored vectors are still comparable with the active model's output.
func (r SettingRepo) EmbeddingModel() (string, error) {
	sett := Setting{}
	err := r.gorm.Table("settings").First(&sett, &Setting{SettingKey: EmbeddingModelSetting}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return legacyEmbeddingModel, nil
	}
	if err != nil {
		return "", err
	}
	return sett.SettingValue, nil
}

// SetEmbeddingModel records the model that produced the stored frame vectors.
func (r SettingRepo) SetEmbeddingModel(modelName string) error {
	return r.save(&Setting{
		SettingKey:   EmbeddingModelSetting,
		SettingValue: modelName,
		SettingType:  "string",
	})
}

// init seeds the rows every install needs.
func (r SettingRepo) init() error {
	return r.gorm.FirstOrCreate(
		&Setting{SettingKey: ReqInterval, SettingValue: "15", SettingType: "int"}).Error
}

// save upserts: SettingKey is the primary key, so an existing row is updated rather
// than duplicated.
func (r SettingRepo) save(setting *Setting) error {
	result := r.gorm.Save(setting)
	if result.Error != nil {
		log.Errorf("[Save] Error saving setting: %s", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Infof("[Save] Setting saved: %s = %s", setting.SettingKey, setting.SettingValue)
	}

	return nil
}
