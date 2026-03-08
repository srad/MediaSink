package services

import (
	"os"
	"path/filepath"

	"github.com/astaxie/beego/utils"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/analysis/detectors/onnx"
	"github.com/srad/mediasink/conf"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"
)

func StartUpJobs() {
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
	StartImport()
	go fixOrphanedFiles()
	go enqueueUnanalyzedRecordings()
}

func deleteOrphanedRecordings() error {
	recordings, err := database.RecordingsList()
	if err != nil {
		return err
	}

	for _, recording := range recordings {
		filePath := recording.ChannelName.AbsoluteChannelFilePath(recording.Filename)
		if !utils.FileExists(filePath) {
			recording.DestroyRecording()
		}
	}

	return nil
}

func deleteChannels() error {
	channels, err := database.ChannelList()
	if err != nil {
		return err
	}

	for _, channel := range channels {
		if channel.Deleted {
			log.Infof("[DeleteChannels] Deleting channel : %s", channel.ChannelName)
			database.DestroyChannel(channel.ChannelID)
		}
	}

	return nil
}

// fixOrphanedFiles Scans the recording folder and checks if an un-imported file is found on the disk.
// Only uncorrupted files will be imported.
func fixOrphanedFiles() error {
	log.Infoln("Fixing orphaned channels ...")

	// 1. Check if channel exists, otherwise delete.
	channels, err := database.ChannelList()
	if err != nil {
		log.Errorf("[FixOrphanedFiles] ChannelList error: %s", err)
		return err
	}
	for _, channel := range channels {
		if !channel.FolderExists() {
			database.DestroyChannel(channel.ChannelID)
		}
	}

	// 2. Check if recording file within channel exists, otherwise destroy.
	log.Infoln("Fixing orphaned recordings ...")
	recordings, err := database.RecordingsList()

	if err != nil {
		log.Errorf("[FixOrphanedFiles] ChannelList error: %s", err)
		return err
	}

	for _, recording := range recordings {
		log.Infof("Handling channel file %s", recording.AbsoluteChannelFilepath())
		err := helpers.CheckVideo(recording.AbsoluteChannelFilepath())
		if err != nil {
			log.Errorf("The file '%s' is corrupted, deleting from disk ... ", recording.Filename)
			if err := recording.DestroyRecording(); err != nil {
				log.Errorf("Deleted file '%s'", recording.Filename)
			}
		}
	}

	return nil
}

// resetOrphanedJobs resets any jobs that were left active (active=true, status=open)
// from a previous run that crashed or was killed. Without this they would be
// stuck forever because the job processor only picks up active=false jobs.
func resetOrphanedJobs() error {
	result := database.DB.Model(&database.Job{}).
		Where("status = ? AND active = ?", database.StatusJobOpen, true).
		Updates(map[string]interface{}{"active": false})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Infof("[StartUpJobs] Reset %d orphaned job(s) to inactive so they will be retried", result.RowsAffected)
	}
	return nil
}

// enqueueUnanalyzedRecordings enqueues analysis jobs for any recordings that do not
// yet have stored frame vectors. Only runs if ONNX runtime and a model are available.
func enqueueUnanalyzedRecordings() {
	if err := onnx.EnsureInitialized(); err != nil {
		log.Infof("[StartUpJobs] ONNX not available, skipping auto-analysis: %v", err)
		return
	}
	if _, err := onnx.GetModelPath("mobilenet_v3_large"); err != nil {
		log.Infof("[StartUpJobs] ONNX model not found, skipping auto-analysis: %v", err)
		return
	}

	analyzedIDs, err := database.ListRecordingIDsWithFrameVectors(1000000)
	if err != nil {
		log.Errorf("[StartUpJobs] Failed to list analyzed recordings: %v", err)
		return
	}
	analyzedSet := make(map[database.RecordingID]struct{}, len(analyzedIDs))
	for _, id := range analyzedIDs {
		analyzedSet[id] = struct{}{}
	}

	recordings, err := database.RecordingsList()
	if err != nil {
		log.Errorf("[StartUpJobs] Failed to list recordings for auto-analysis: %v", err)
		return
	}

	enqueued := 0
	for _, rec := range recordings {
		if _, done := analyzedSet[rec.RecordingID]; done {
			continue
		}
		if _, err := rec.EnqueueAnalysisJob(); err == nil {
			enqueued++
		}
	}
	if enqueued > 0 {
		log.Infof("[StartUpJobs] Enqueued %d analysis job(s) for unanalyzed recordings", enqueued)
	} else {
		log.Infof("[StartUpJobs] All recordings already analyzed")
	}
}

// cleanupDeprecatedPreviewArtifacts removes old preview folders and files that
// have been replaced by the new frames-based preview system.
func cleanupDeprecatedPreviewArtifacts() {
	cfg := conf.Read()
	channels, err := database.ChannelList()
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
