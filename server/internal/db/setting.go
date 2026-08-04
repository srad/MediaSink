package db

import (
	"context"
	"errors"
	"fmt"

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

// SettingStore is the settings aggregate. It always carries the connection it was built
// with, which matters more here than elsewhere: Migrate reads and seeds settings while
// running against a database that is not necessarily the one the package global points
// at.
type SettingStore struct {
	gorm *gorm.DB
}

func NewSettingStore(conn *gorm.DB) *SettingStore {
	return &SettingStore{gorm: conn}
}

// EmbeddingModel returns the model that produced the stored frame vectors.
// A missing row means the database predates the setting, so it reports the
// legacy model rather than an error — callers use this to decide whether the
// stored vectors are still comparable with the active model's output.
func (s *SettingStore) EmbeddingModel(ctx context.Context) (string, error) {
	sett := Setting{}
	err := s.gorm.WithContext(ctx).
		Table("settings").
		First(&sett, &Setting{SettingKey: EmbeddingModelSetting}).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return legacyEmbeddingModel, nil
	}
	if err != nil {
		return "", fmt.Errorf("read the %s setting: %w", EmbeddingModelSetting, err)
	}
	return sett.SettingValue, nil
}

// SetEmbeddingModel records the model that produced the stored frame vectors.
func (s *SettingStore) SetEmbeddingModel(ctx context.Context, modelName string) error {
	return s.save(ctx, &Setting{
		SettingKey:   EmbeddingModelSetting,
		SettingValue: modelName,
		SettingType:  "string",
	})
}

// init seeds the rows every install needs.
func (s *SettingStore) init(ctx context.Context) error {
	if err := s.gorm.WithContext(ctx).FirstOrCreate(
		&Setting{SettingKey: ReqInterval, SettingValue: "15", SettingType: "int"}).Error; err != nil {
		return fmt.Errorf("seed the %s setting: %w", ReqInterval, err)
	}
	return nil
}

// save upserts: SettingKey is the primary key, so an existing row is updated rather
// than duplicated.
func (s *SettingStore) save(ctx context.Context, setting *Setting) error {
	result := s.gorm.WithContext(ctx).Save(setting)
	if result.Error != nil {
		log.Errorf("[Save] Error saving setting: %s", result.Error)
		return fmt.Errorf("save setting %s: %w", setting.SettingKey, result.Error)
	}

	if result.RowsAffected > 0 {
		log.Infof("[Save] Setting saved: %s = %s", setting.SettingKey, setting.SettingValue)
	}

	return nil
}
