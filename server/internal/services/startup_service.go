package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/analysis/detectors/onnx"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/store/vector"
	"github.com/srad/mediasink/server/internal/util"
)

func StartUpJobs(settings *db.SettingStore) {
	log.Infoln("[StartUpJobs] Running startup job ...")

	if err := resetOrphanedJobs(); err != nil { // Blocking — must run before job processor starts
		log.Errorf("[StartUpJobs] Failed to reset orphaned jobs: %v", err)
	}
	if err := deleteChannels(); err != nil { // Blocking
		log.Errorf("[DeleteChannels] ChannelList error: %s", err)
	}
	if err := deleteOrphanedRecordings(); err != nil { // Blocking
		log.Errorln(err)
	}
	cleanupDeprecatedPreviewArtifacts() // Clean up old preview artifacts
	if !StartImport() {
		log.Infoln("[StartUpJobs] Import already running, skipping startup import")
	}
	go func() {
		if err := fixOrphanedFiles(); err != nil {
			log.Errorf("[StartUpJobs] fixOrphanedFiles failed: %s", err)
		}
	}()
	go enqueueUnanalyzedRecordings(settings)
}

func deleteOrphanedRecordings() error {
	recordings, err := db.RecordingsList()
	if err != nil {
		return err
	}

	for _, recording := range recordings {
		filePath := recording.ChannelName.AbsoluteChannelFilePath(recording.Filename)
		if !util.FileExists(filePath) {
			if err := recording.DestroyRecording(); err != nil {
				log.Errorf("[DeleteOrphanedRecordings] Error destroying recording '%s': %s", recording.Filename, err)
			}
		}
	}

	return nil
}

func deleteChannels() error {
	channels, err := db.ChannelList()
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.Deleted {
			log.Infof("[DeleteChannels] Deleting channel : %s", channel.ChannelName)
			if err := db.DestroyChannel(channel.ChannelID); err != nil {
				log.Errorf("[DeleteChannels] Error destroying channel '%s': %s", channel.ChannelName, err)
			}
		}
	}

	return nil
}

// fixOrphanedFiles Scans the recording folder and checks if an un-imported file is found on the disk.
// Only uncorrupted files will be imported.
func fixOrphanedFiles() error {
	log.Infoln("Fixing orphaned channels ...")

	// 1. Check if channel exists, otherwise delete.
	channels, err := db.ChannelList()
	if err != nil {
		log.Errorf("[FixOrphanedFiles] ChannelList error: %s", err)
		return err
	}
	for _, channel := range channels {
		if !channel.FolderExists() {
			if err := db.DestroyChannel(channel.ChannelID); err != nil {
				log.Errorf("[FixOrphanedFiles] Error destroying channel '%s': %s", channel.ChannelName, err)
			}
		}
	}

	// 2. Check if recording file within channel exists, otherwise destroy.
	log.Infoln("Fixing orphaned recordings ...")
	recordings, err := db.RecordingsList()

	if err != nil {
		log.Errorf("[FixOrphanedFiles] ChannelList error: %s", err)
		return err
	}

	for _, recording := range recordings {
		log.Infof("Handling channel file %s", recording.AbsoluteChannelFilepath())
		err := util.CheckVideo(recording.AbsoluteChannelFilepath())

		if !isCorruptionEvidence(err) {
			if err != nil {
				log.Warnf("Integrity check for '%s' was interrupted, leaving the file untouched", recording.Filename)
			}
			continue
		}

		log.Errorf("The file '%s' is corrupted, deleting from disk ... ", recording.Filename)
		if errDestroy := recording.DestroyRecording(); errDestroy != nil {
			log.Errorf("Error deleting file '%s': %s", recording.Filename, errDestroy)
		} else {
			log.Infof("Deleted file '%s'", recording.Filename)
		}
	}

	return nil
}

// isCorruptionEvidence reports whether a CheckVideo failure is evidence that the
// file itself is bad, and therefore safe to act on by deleting the recording.
//
// A deliberately interrupted check says nothing about the file: util.Interrupt
// can reach any registered process, including this one, and treating that as
// corruption would destroy a healthy recording along with its database row.
func isCorruptionEvidence(err error) bool {
	return err != nil && !errors.Is(err, util.ErrInterrupted)
}

// resetOrphanedJobs resets any jobs that were left active (active=true, status=open)
// from a previous run that crashed or was killed. Without this they would be
// stuck forever because the job processor only picks up active=false jobs.
func resetOrphanedJobs() error {
	result := db.DB.Model(&db.Job{}).
		Where("status = ? AND active = ?", db.StatusJobOpen, true).
		Updates(map[string]interface{}{"active": false})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Infof("[StartUpJobs] Reset %d orphaned job(s) to inactive so they will be retried", result.RowsAffected)
	}
	return nil
}

// enqueueUnanalyzedRecordings decides the next deterministic step for each recording.
// Missing/invalid previews enqueue preview generation. Valid previews without stored
// vectors enqueue analysis. Fully analyzed recordings enqueue nothing.
func enqueueUnanalyzedRecordings(settings *db.SettingStore) {
	if err := onnx.EnsureInitialized(); err != nil {
		log.Infof("[StartUpJobs] ONNX not available, skipping auto-analysis: %v", err)
		return
	}
	if _, err := onnx.GetModelPath(onnx.DefaultModelName); err != nil {
		log.Infof("[StartUpJobs] ONNX model not found, skipping auto-analysis: %v", err)
		return
	}

	analyzedIDs, err := vector.Default().ListRecordingIDs(context.Background(), 1000000)
	if err != nil {
		log.Errorf("[StartUpJobs] Failed to list analyzed recordings: %v", err)
		return
	}
	analyzedSet := make(map[db.RecordingID]struct{}, len(analyzedIDs))
	for _, id := range analyzedIDs {
		analyzedSet[id] = struct{}{}
	}

	recordings, err := db.RecordingsList()
	if err != nil {
		log.Errorf("[StartUpJobs] Failed to list recordings for auto-analysis: %v", err)
		return
	}

	previewJobs := 0
	analysisJobs := 0
	for _, rec := range recordings {
		if _, done := analyzedSet[rec.RecordingID]; done {
			continue
		}

		task, err := EnqueueAnalysisPipeline(rec)
		if err != nil {
			continue
		}

		switch task {
		case db.TaskPreviewFrames:
			previewJobs++
		case db.TaskAnalyzeFrames:
			analysisJobs++
		}
	}
	log.Infof("[StartUpJobs] Enqueued %d preview job(s) and %d analysis job(s) during startup backfill", previewJobs, analysisJobs)

	// Record the model only once the backfill has been queued. If the process dies
	// before this, the next boot sees the old value, drops frame_vectors again (it
	// holds nothing worth keeping at that point) and re-queues the backfill.
	if err := settings.SetEmbeddingModel(context.Background(), onnx.DefaultModelName); err != nil {
		log.Errorf("[StartUpJobs] Failed to record the active embedding model: %v", err)
	}
}

// cleanupDeprecatedPreviewArtifacts removes old preview folders and files that
// have been replaced by the new frames-based preview system.
func cleanupDeprecatedPreviewArtifacts() {
	cfg := config.Read()
	channels, err := db.ChannelList()
	if err != nil {
		log.Errorf("[CleanupDeprecatedPreviews] Error getting channel list: %s", err)
		return
	}

	for _, channel := range channels {
		previewsBasePath := filepath.Join(cfg.RecordingsAbsolutePath, channel.ChannelName.String(), cfg.DataPath)
		cleanupDeprecatedPreviewArtifactsIn(previewsBasePath)
	}
}

func cleanupDeprecatedPreviewArtifactsIn(previewsBasePath string) {
	deprecatedFolders := []string{"posters", "stripes", "previews", "montages", "videos"}
	deprecatedFiles := []string{"info.csv"}

	for _, folder := range deprecatedFolders {
		deprecatedPath := filepath.Join(previewsBasePath, folder)
		if _, err := os.Stat(deprecatedPath); err == nil {
			log.Infof("[CleanupDeprecatedPreviews] Removing deprecated preview folder: %s", deprecatedPath)
			if err := os.RemoveAll(deprecatedPath); err != nil {
				log.Errorf("[CleanupDeprecatedPreviews] Error removing %s: %s", deprecatedPath, err)
			} else {
				log.Infof("[CleanupDeprecatedPreviews] Successfully removed: %s", deprecatedPath)
			}
		}
	}

	for _, filename := range deprecatedFiles {
		deprecatedFile := filepath.Join(previewsBasePath, filename)
		if _, err := os.Stat(deprecatedFile); err == nil {
			log.Infof("[CleanupDeprecatedPreviews] Removing deprecated preview file: %s", deprecatedFile)
			if err := os.Remove(deprecatedFile); err != nil {
				log.Errorf("[CleanupDeprecatedPreviews] Error removing %s: %s", deprecatedFile, err)
			} else {
				log.Infof("[CleanupDeprecatedPreviews] Successfully removed: %s", deprecatedFile)
			}
		}
	}
}
