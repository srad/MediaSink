package db

import (
	"path/filepath"
	"testing"

	"github.com/srad/mediasink/server/config"
)

// settingStore opens a throwaway database through the real Open, so the repo runs
// against the schema Migrate produces — which includes the ReqInterval row init()
// seeds.
// settingStore returns the store plus the handle behind it, because a few cases assert
// directly against the rows.
func settingStore(t *testing.T) (*SettingStore, *Handle) {
	t.Helper()

	handle, err := Open(t.Context(), config.Cfg{DbFileName: filepath.Join(t.TempDir(), "settings.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return NewSettingStore(handle.Gorm()), handle
}

func TestSettingStoreEmbeddingModelRoundTrip(t *testing.T) {
	settings, _ := settingStore(t)

	if err := settings.SetEmbeddingModel(t.Context(), "some_model_v2"); err != nil {
		t.Fatalf("SetEmbeddingModel: %v", err)
	}

	got, err := settings.EmbeddingModel(t.Context())
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
func TestSettingStoreEmbeddingModelMissingRowReportsTheLegacyModel(t *testing.T) {
	settings, _ := settingStore(t)

	got, err := settings.EmbeddingModel(t.Context())
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
func TestSettingStoreSaveUpdatesInPlace(t *testing.T) {
	settings, handle := settingStore(t)

	for _, model := range []string{"first_model", "second_model", "third_model"} {
		if err := settings.SetEmbeddingModel(t.Context(), model); err != nil {
			t.Fatalf("SetEmbeddingModel(%q): %v", model, err)
		}
	}

	var rows int64
	if err := handle.Gorm().Model(&Setting{}).
		Where("setting_key = ?", EmbeddingModelSetting).Count(&rows).Error; err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if rows != 1 {
		t.Errorf("embedding_model row count = %d after three writes, want 1", rows)
	}

	got, err := settings.EmbeddingModel(t.Context())
	if err != nil {
		t.Fatalf("EmbeddingModel: %v", err)
	}
	if got != "third_model" {
		t.Errorf("EmbeddingModel = %q, want the last value written", got)
	}
}

// init runs as part of Migrate, so Open must leave the seed row behind.
func TestSettingStoreInitSeedsReqInterval(t *testing.T) {
	_, handle := settingStore(t)

	var seeded Setting
	if err := handle.Gorm().Model(&Setting{}).
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
func TestSettingStoreWritesToItsOwnHandleNotTheGlobal(t *testing.T) {
	first, _ := settingStore(t)
	second, _ := settingStore(t) // Open reassigns the global DB to this one.

	if err := first.SetEmbeddingModel(t.Context(), "model_written_to_first"); err != nil {
		t.Fatalf("SetEmbeddingModel against the first store: %v", err)
	}

	got, err := second.EmbeddingModel(t.Context())
	if err != nil {
		t.Fatalf("EmbeddingModel against the second store: %v", err)
	}
	if got == "model_written_to_first" {
		t.Error("a write through the first store's repo was visible from the second store")
	}
}
