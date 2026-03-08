package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePreviewFrames(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		hasDBEntry bool
		setupPath  func(t *testing.T) string
		wantNeeds  bool
		wantReason string
		wantErr    bool
	}

	tests := []testCase{
		{
			name:       "valid preview",
			hasDBEntry: true,
			setupPath: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeEmptyFile(t, filepath.Join(dir, "0.jpg"))
				writeEmptyFile(t, filepath.Join(dir, "2.jpg"))
				return dir
			},
			wantNeeds:  false,
			wantReason: "",
		},
		{
			name:       "missing db row",
			hasDBEntry: false,
			setupPath: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeEmptyFile(t, filepath.Join(dir, "0.jpg"))
				writeEmptyFile(t, filepath.Join(dir, "2.jpg"))
				return dir
			},
			wantNeeds:  true,
			wantReason: previewValidationMissingDBRow,
		},
		{
			name:       "missing folder",
			hasDBEntry: true,
			setupPath: func(t *testing.T) string {
				t.Helper()
				return filepath.Join(t.TempDir(), "does-not-exist")
			},
			wantNeeds:  true,
			wantReason: previewValidationMissingFolder,
		},
		{
			name:       "legacy file names only",
			hasDBEntry: true,
			setupPath: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeEmptyFile(t, filepath.Join(dir, "frame-000001.jpg"))
				writeEmptyFile(t, filepath.Join(dir, "frame-000002.jpg"))
				return dir
			},
			wantNeeds:  true,
			wantReason: previewValidationInvalidFormat,
		},
		{
			name:       "non timestamp jpg names",
			hasDBEntry: true,
			setupPath: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeEmptyFile(t, filepath.Join(dir, "abc.jpg"))
				writeEmptyFile(t, filepath.Join(dir, "cover.jpg"))
				return dir
			},
			wantNeeds:  true,
			wantReason: previewValidationInvalidFormat,
		},
		{
			name:       "insufficient timestamp frames",
			hasDBEntry: true,
			setupPath: func(t *testing.T) string {
				t.Helper()
				dir := t.TempDir()
				writeEmptyFile(t, filepath.Join(dir, "0.jpg"))
				return dir
			},
			wantNeeds:  true,
			wantReason: previewValidationInsufficient,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			path := tc.setupPath(t)
			needs, reason, err := validatePreviewFrames(path, tc.hasDBEntry)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err mismatch: got err=%v, wantErr=%v", err, tc.wantErr)
			}
			if needs != tc.wantNeeds {
				t.Fatalf("needs mismatch: got %v, want %v", needs, tc.wantNeeds)
			}
			if reason != tc.wantReason {
				t.Fatalf("reason mismatch: got %q, want %q", reason, tc.wantReason)
			}
		})
	}
}

func writeEmptyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}
