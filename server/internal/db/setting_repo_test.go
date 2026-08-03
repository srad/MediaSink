package db

import (
	"path/filepath"
	"testing"

	"github.com/srad/mediasink/server/config"
)

// settingStore opens a throwaway database through the real Open, so the repo runs
// against the schema Migrate produces — which includes the ReqInterval row init()
// seeds.
func settingStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(config.Cfg{DbFileName: filepath.Join(t.TempDir(), "settings.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func TestSettingRepoEmbeddingModelRoundTrip(t *testing.T) {
	settings := settingStore(t).Settings()

	if err := settings.SetEmbeddingModel("some_model_v2"); err != nil {
		t.Fatalf("SetEmbeddingModel: %v", err)
	}

	got, err := settings.EmbeddingModel()
	if err != nil {
		t.Fatalf("EmbeddingModel: %v", err)
	}
	if got != "some_model_v2" {
		t.Errorf("EmbeddingModel = %q, want %q", got, "some_model_v2")
	}
}

// A database that predates the setting holds vectors from the legacy model, so a
// missing row must report that model rather than an error — dropStaleFrameVectors
// treats an error as "cannot tell" and leaves stale vectors in place.
func TestSettingRepoEmbeddingModelMissingRowReportsTheLegacyModel(t *testing.T) {
	settings := settingStore(t).Settings()

	got, err := settings.EmbeddingModel()
	if err != nil {
		t.Fatalf("EmbeddingModel on a database with no such row: %v", err)
	}
	if got != legacyEmbeddingModel {
		t.Errorf("EmbeddingModel = %q, want the legacy model %q", got, legacyEmbeddingModel)
	}
}

// SettingKey is the primary key, so save must update in place. If it inserted instead,
// the row count would climb on every boot and EmbeddingModel would start returning
// whichever row First happened to pick.
func TestSettingRepoSaveUpdatesInPlace(t *testing.T) {
	store := settingStore(t)
	settings := store.Settings()

	for _, model := range []string{"first_model", "second_model", "third_model"} {
		if err := settings.SetEmbeddingModel(model); err != nil {
			t.Fatalf("SetEmbeddingModel(%q): %v", model, err)
		}
	}

	var rows int64
	if err := store.gorm.Model(&Setting{}).
		Where("setting_key = ?", EmbeddingModelSetting).Count(&rows).Error; err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if rows != 1 {
		t.Errorf("embedding_model row count = %d after three writes, want 1", rows)
	}

	got, err := settings.EmbeddingModel()
	if err != nil {
		t.Fatalf("EmbeddingModel: %v", err)
	}
	if got != "third_model" {
		t.Errorf("EmbeddingModel = %q, want the last value written", got)
	}
}

// init runs as part of Migrate, so Open must leave the seed row behind.
func TestSettingRepoInitSeedsReqInterval(t *testing.T) {
	store := settingStore(t)

	var seeded Setting
	if err := store.gorm.Model(&Setting{}).
		Where("setting_key = ?", ReqInterval).First(&seeded).Error; err != nil {
		t.Fatalf("ReqInterval was not seeded by Migrate: %v", err)
	}
	if seeded.SettingValue != "15" {
		t.Errorf("ReqInterval = %q, want %q", seeded.SettingValue, "15")
	}
}

// The same scoping guarantee 2b.1 proved for Migrate, now for the repo the migration
// goes through. Open reassigns the package global, so a repo reaching for DB instead
// of its own handle would write into the most recently opened database.
func TestSettingRepoWritesToItsOwnHandleNotTheGlobal(t *testing.T) {
	first := settingStore(t)
	second := settingStore(t) // Open reassigns the global DB to this one.

	if err := first.Settings().SetEmbeddingModel("model_written_to_first"); err != nil {
		t.Fatalf("SetEmbeddingModel against the first store: %v", err)
	}

	got, err := second.Settings().EmbeddingModel()
	if err != nil {
		t.Fatalf("EmbeddingModel against the second store: %v", err)
	}
	if got == "model_written_to_first" {
		t.Error("a write through the first store's repo was visible from the second store")
	}
}
