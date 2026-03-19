package services

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
)

// EnqueueAnalysisPipeline queues the next valid step for a recording.
// If previews are missing or invalid, it queues preview generation first.
// Otherwise it queues frame analysis directly.
func EnqueueAnalysisPipeline(recording *db.Recording) (db.JobTask, error) {
	if recording == nil {
		return "", fmt.Errorf("recording is nil")
	}

	previewState, validationErr := ValidateRecordingPreview(recording)
	if validationErr != nil {
		log.Warnf("[EnqueueAnalysisPipeline] Preview validation for recording %d returned %v", recording.RecordingID, validationErr)
	}

	if previewState.NeedsRegeneration {
		_, err := recording.EnqueuePreviewFramesJob()
		return db.TaskPreviewFrames, err
	}

	_, err := recording.EnqueueAnalysisJob()
	return db.TaskAnalyzeFrames, err
}
