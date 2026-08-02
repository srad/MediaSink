package db

import (
	"os"
	"testing"

	"github.com/srad/mediasink/server/internal/analysis/detectors/onnx"
)

// frameVectorsExists reports whether the vec0 table is present.
func frameVectorsExists(t *testing.T) bool {
	t.Helper()
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("DB handle: %v", err)
	}
	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='frame_vectors'`).Scan(&count); err != nil {
		t.Fatalf("sqlite_master query: %v", err)
	}
	return count > 0
}

// setupEmbeddingModelDB prepares a DB with the settings table and a live
// frame_vectors table, standing in for an install that has already analyzed.
func setupEmbeddingModelDB(t *testing.T) {
	t.Helper()
	setupVecDB(t)

	// Every ":memory:" connection is its own private database, so pin the pool to
	// one connection — otherwise the table created here and the check afterwards
	// can land on different, unrelated databases.
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("DB handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)

	if err := DB.AutoMigrate(&Setting{}); err != nil {
		t.Fatalf("AutoMigrate(Setting): %v", err)
	}
	if err := ensureVecTable(testDim); err != nil {
		t.Fatalf("ensureVecTable: %v", err)
	}
	if !frameVectorsExists(t) {
		t.Fatal("precondition failed: frame_vectors was not created")
	}
}

// A model swap must drop the stored vectors. Their dimension is unchanged, so
// nothing else would catch it — the old and new embeddings would simply be
// compared against each other and rank as noise.
func TestDropStaleFrameVectors_ModelChanged(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupEmbeddingModelDB(t)

	if err := SetEmbeddingModel("some_other_model"); err != nil {
		t.Fatalf("SetEmbeddingModel: %v", err)
	}

	dropStaleFrameVectors()

	if frameVectorsExists(t) {
		t.Error("frame_vectors survived an embedding model change")
	}
}

// A database predating the setting holds mobilenet_v3_large vectors, so a
// missing row must be treated as stale rather than as "already current".
func TestDropStaleFrameVectors_MissingSetting(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupEmbeddingModelDB(t)

	dropStaleFrameVectors()

	if frameVectorsExists(t) {
		t.Error("frame_vectors survived with no recorded embedding model")
	}
}

// The common case: same model as last boot, so the vectors must be left alone.
// Getting this wrong would silently wipe the analysis on every restart.
func TestDropStaleFrameVectors_ModelUnchanged(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupEmbeddingModelDB(t)

	if err := SetEmbeddingModel(onnx.DefaultModelName); err != nil {
		t.Fatalf("SetEmbeddingModel: %v", err)
	}

	dropStaleFrameVectors()

	if !frameVectorsExists(t) {
		t.Error("frame_vectors was dropped even though the model is unchanged")
	}
}

func TestGetEmbeddingModel_RoundTrip(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)
	if err := DB.AutoMigrate(&Setting{}); err != nil {
		t.Fatalf("AutoMigrate(Setting): %v", err)
	}

	got, err := GetEmbeddingModel()
	if err != nil {
		t.Fatalf("GetEmbeddingModel on empty settings: %v", err)
	}
	if got != legacyEmbeddingModel {
		t.Errorf("missing setting: got %q, want %q", got, legacyEmbeddingModel)
	}

	if err := SetEmbeddingModel(onnx.DefaultModelName); err != nil {
		t.Fatalf("SetEmbeddingModel: %v", err)
	}
	if got, err = GetEmbeddingModel(); err != nil {
		t.Fatalf("GetEmbeddingModel: %v", err)
	}
	if got != onnx.DefaultModelName {
		t.Errorf("after write: got %q, want %q", got, onnx.DefaultModelName)
	}
}
