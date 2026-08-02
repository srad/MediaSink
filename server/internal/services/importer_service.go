package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/util"
)

var (
	importMu       sync.Mutex
	importing      = false
	importSize     int
	importProgress int
)

// StartImport begins an import unless one is already running. It reports
// whether a new import was started, so callers can surface a conflict rather
// than silently stacking concurrent scans over the same tree.
func StartImport() bool {
	importMu.Lock()
	defer importMu.Unlock()

	if importing {
		return false
	}
	importing = true
	importSize = 0
	importProgress = 0

	go runImport()
	return true
}

func IsImporting() bool {
	importMu.Lock()
	defer importMu.Unlock()
	return importing
}

func GetImportProgress() (int, int) {
	importMu.Lock()
	defer importMu.Unlock()
	return importProgress, importSize
}

// setImportTotal records how many channel folders the scan will visit.
func setImportTotal(size int) {
	importMu.Lock()
	defer importMu.Unlock()
	importSize = size
	importProgress = 0
}

// advanceImportProgress increments the visited-folder counter and returns the
// new value, so callers can log progress without touching shared state.
func advanceImportProgress() int {
	importMu.Lock()
	defer importMu.Unlock()
	importProgress++
	return importProgress
}

func runImport() {
	defer func() {
		importMu.Lock()
		importing = false
		importMu.Unlock()
	}()

	if err := ImportChannels(context.Background()); err != nil {
		log.Errorf("Error during channel import: %v", err)
	}
}

// ImportChannels Imports folders and videos found on disk.
//
// 1. Import all folders as channels found in the recording path.
// 2. If the folder contains the channel.json backup file, then reconstruct the channel information from this file.
// 3. Then search on each folder for media files to import as recordings.
// 4. If the recordings do not contain previews, their creation will be scheduled.
func ImportChannels(ctx context.Context) error {
	cfg := config.Read()

	log.Infoln("------------------------------------------------------------------------------------------")
	log.Infof("Scanning file system for media: %s", cfg.RecordingsAbsolutePath)
	log.Infoln("------------------------------------------------------------------------------------------")

	recordingFolder, err := os.Open(cfg.RecordingsAbsolutePath)
	if err != nil {
		return fmt.Errorf("failed opening directory '%s': %w", cfg.RecordingsAbsolutePath, err)
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			log.Errorf("error closing folder %s: %v", file.Name(), err)
		}
	}(recordingFolder)

	// ---------------------------------------------------------------------------------
	// Traverse folders (channels)
	// ---------------------------------------------------------------------------------
	channelFolders, err := recordingFolder.Readdirnames(0)
	if err != nil {
		return fmt.Errorf("error reading directory entries from '%s': %w", cfg.RecordingsAbsolutePath, err)
	}
	total := len(channelFolders)
	setImportTotal(total)
	for i, name := range channelFolders {
		if err := ctx.Err(); err != nil {
			log.Infof("[Import] Cancelled after %d/%d channels", i, total)
			return err
		}

		progress := advanceImportProgress()
		channelName := db.ChannelName(name)
		// Is no directory, skip
		if dir, err := os.Stat(channelName.AbsoluteChannelPath()); err != nil || !dir.IsDir() {
			continue
		}

		newChannel, errCreate := db.CreateChannel(channelName, channelName.String(), "https://"+channelName.String())
		if errCreate != nil {
			log.Errorf("[Import/%s (%d/%d)] Error creating channel: %v", channelName, progress, total, errCreate)
			// Skip this channel if it cannot be created.
			continue
		}

		// ---------------------------------------------------------------------------------
		// Import individual files
		// ---------------------------------------------------------------------------------
		files, errReadDir := os.ReadDir(channelName.AbsoluteChannelPath())
		if errReadDir != nil {
			log.Errorf("[Import/%s] Error reading: %s", channelName, errReadDir)
			continue
		}

		// ---------------------------------------------------------------------------------
		// Traverse all mp4 files and add to models if not existent
		// ---------------------------------------------------------------------------------
		j := 0
		log.Infof("[Import/%s (%d/%d)] Traverse all mp4 files and add to models if not existent (files: %d) ...", channelName, progress, total, len(files))
		for _, file := range files {
			j++
			mp4File := !file.IsDir() && filepath.Ext(file.Name()) == ".mp4"
			if !mp4File {
				continue
			}

			log.Debugf("[Import/%s (%d/%d) (%d/%d)] Checking file: %s", channelName, progress, total, j, len(files), file.Name())

			filename := db.RecordingFileName(file.Name())
			video := &util.Video{FilePath: channelName.AbsoluteChannelFilePath(filename)}

			if _, errVideoInfo := video.GetVideoInfo(); errVideoInfo != nil {
				log.Errorf("[Import/%s] File '%s' seems corrupted, deleting: %s", channelName, file.Name(), errVideoInfo)
				if errDestroy := db.DeleteRecordingData(channelName, filename); errDestroy != nil {
					log.Errorf("[Import/%s] Error deleting: %s: %s", channelName, file.Name(), errDestroy)
				} else {
					log.Infof("[Import/%s] Deleted: %s", channelName, file.Name())
				}
				continue
			}

			// File seems ok, try to add.
			newRecording, errAdd := db.AddIfNotExists(newChannel.ChannelID, newChannel.ChannelName, db.RecordingFileName(file.Name()))
			if errAdd != nil {
				log.Errorf("[Import/%s] Error: %s", channelName, errAdd)
				continue
			}

			previewFramesPath := newRecording.RecordingID.GetPreviewFramesPath(newRecording.ChannelName)
			_, dbErr := db.FindVideoPreviewByRecordingID(newRecording.RecordingID)
			hasDBEntry := dbErr == nil

			needsPreview, reason, errValidate := validatePreviewFrames(previewFramesPath, hasDBEntry)
			if errValidate != nil {
				log.Errorf("[Import/%s] Preview validation error for recording %d: %v", channelName, newRecording.RecordingID, errValidate)
				needsPreview = true
				if reason == "" {
					reason = previewValidationUnknown
				}
			}

			if needsPreview {
				log.Infof("[Import/%s] Preview regeneration required for recording %d (%s)", channelName, newRecording.RecordingID, reason)
				if _, err := newRecording.EnqueuePreviewFramesJob(); err != nil {
					log.Errorf("[Import/%s] Error enqueueing preview frames job: %s", channelName, err)
				}
			}
		}
	}

	return nil
}
