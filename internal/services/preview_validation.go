package services

import (
	"fmt"
	"os"
	"regexp"

	"github.com/srad/mediasink/internal/db"
)

const (
	previewValidationMissingDBRow    = "missing_db_row"
	previewValidationMissingFolder   = "missing_folder"
	previewValidationInvalidFormat   = "invalid_frame_format"
	previewValidationInsufficient    = "insufficient_frames"
	previewValidationFolderReadError = "folder_read_error"
	previewValidationUnknown         = "unknown"
	minRequiredPreviewFrameCount     = 2
)

var timestampFramePattern = regexp.MustCompile(`^\d+\.jpg$`)

func isTimestampFrameFile(name string) bool {
	return timestampFramePattern.MatchString(name)
}

func validatePreviewFrames(previewFramesPath string, hasDBEntry bool) (needsRegeneration bool, reason string, err error) {
	if !hasDBEntry {
		return true, previewValidationMissingDBRow, nil
	}

	entries, readErr := os.ReadDir(previewFramesPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return true, previewValidationMissingFolder, nil
		}
		return true, previewValidationFolderReadError, readErr
	}

	fileCount := 0
	validCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileCount++
		if isTimestampFrameFile(entry.Name()) {
			validCount++
		}
	}

	if validCount == 0 {
		if fileCount > 0 {
			return true, previewValidationInvalidFormat, nil
		}
		return true, previewValidationInsufficient, nil
	}

	if validCount < minRequiredPreviewFrameCount {
		return true, previewValidationInsufficient, nil
	}

	return false, "", nil
}

type PreviewValidationResult struct {
	NeedsRegeneration bool   `json:"needsRegeneration"`
	Reason            string `json:"reason,omitempty"`
}

func (r PreviewValidationResult) IsValid() bool {
	return !r.NeedsRegeneration
}

func ValidateRecordingPreview(recording *db.Recording) (PreviewValidationResult, error) {
	if recording == nil {
		return PreviewValidationResult{}, fmt.Errorf("recording is nil")
	}

	needsRegeneration, reason, err := validatePreviewFrames(
		recording.RecordingID.GetPreviewFramesPath(recording.ChannelName),
		recording.VideoPreviews != nil,
	)
	if err != nil {
		return PreviewValidationResult{
			NeedsRegeneration: needsRegeneration,
			Reason:            reason,
		}, err
	}

	return PreviewValidationResult{
		NeedsRegeneration: needsRegeneration,
		Reason:            reason,
	}, nil
}
