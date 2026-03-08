package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupDeprecatedPreviewArtifactsIn(t *testing.T) {
	t.Parallel()

	base := t.TempDir()

	deprecatedFolders := []string{"posters", "stripes", "previews", "montages", "videos"}
	for _, folder := range deprecatedFolders {
		p := filepath.Join(base, folder)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
		if err := os.WriteFile(filepath.Join(p, "sample.txt"), []byte("legacy"), 0o644); err != nil {
			t.Fatalf("write legacy file in %s: %v", p, err)
		}
	}

	if err := os.WriteFile(filepath.Join(base, "info.csv"), []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write info.csv: %v", err)
	}

	framesDir := filepath.Join(base, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatalf("mkdir frames: %v", err)
	}
	if err := os.WriteFile(filepath.Join(framesDir, "0.jpg"), []byte("frame"), 0o644); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	if err := os.WriteFile(filepath.Join(base, "live.jpg"), []byte("live"), 0o644); err != nil {
		t.Fatalf("write live.jpg: %v", err)
	}

	cleanupDeprecatedPreviewArtifactsIn(base)

	for _, folder := range deprecatedFolders {
		assertNotExists(t, filepath.Join(base, folder))
	}
	assertNotExists(t, filepath.Join(base, "info.csv"))

	assertExists(t, framesDir)
	assertExists(t, filepath.Join(framesDir, "0.jpg"))
	assertExists(t, filepath.Join(base, "live.jpg"))
}

func TestCleanupDeprecatedPreviewArtifactsIn_Idempotent(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	framesDir := filepath.Join(base, "frames")
	if err := os.MkdirAll(framesDir, 0o755); err != nil {
		t.Fatalf("mkdir frames: %v", err)
	}

	cleanupDeprecatedPreviewArtifactsIn(base)
	cleanupDeprecatedPreviewArtifactsIn(base)

	assertExists(t, framesDir)
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to not exist, err=%v", path, err)
	}
}
