package services

import (
	"os"
	"regexp"
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
