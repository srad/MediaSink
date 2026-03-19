package services

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPreviewFrames_SortsNumerically(t *testing.T) {
	dir := t.TempDir()
	for _, ts := range []int{100, 0, 10, 20} {
		if err := writeTestFrame(dir, ts, color.RGBA{R: 255, A: 255}); err != nil {
			t.Fatalf("writeTestFrame: %v", err)
		}
	}

	frames, err := LoadPreviewFrames(dir)
	if err != nil {
		t.Fatalf("LoadPreviewFrames: %v", err)
	}

	want := []uint64{0, 10, 20, 100}
	if len(frames) != len(want) {
		t.Fatalf("expected %d frames, got %d", len(want), len(frames))
	}

	for index, expected := range want {
		if frames[index].Timestamp != expected {
			t.Fatalf("frames[%d].Timestamp: want %d, got %d", index, expected, frames[index].Timestamp)
		}
	}
}

func TestLoadPreviewFrames_EmptyDir(t *testing.T) {
	frames, err := LoadPreviewFrames(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(frames) != 0 {
		t.Fatalf("expected no frames, got %d", len(frames))
	}
}

func TestLoadPreviewFrames_InvalidFormatInNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "frame-000001.jpg"), []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := LoadPreviewFrames(dir); err == nil {
		t.Fatal("expected error for non-empty directory without timestamp files")
	}
}
