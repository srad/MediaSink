package services

import (
	"fmt"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/ws"
)

// RegenerationProgress represents the current state of preview regeneration
type RegenerationProgress struct {
	Current      int    `json:"current"`
	Total        int    `json:"total"`
	CurrentVideo string `json:"currentVideo"`
	IsRunning    bool   `json:"isRunning"`
}

// PreviewRegenerator manages preview frame regeneration with thread-safe access
type PreviewRegenerator struct {
	mu      sync.RWMutex
	current int
	total   int
	video   string
	running bool
}

// NewPreviewRegenerator creates a new PreviewRegenerator instance
func NewPreviewRegenerator() *PreviewRegenerator {
	return &PreviewRegenerator{
		running: false,
	}
}

// Start initializes a regeneration session with the total number of videos
func (pr *PreviewRegenerator) Start(total int) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.running = true
	pr.total = total
	pr.current = 0
	pr.video = ""
}

// Update updates the current progress
func (pr *PreviewRegenerator) Update(current int, video string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.current = current
	pr.video = video
}

// Stop marks the regeneration as complete
func (pr *PreviewRegenerator) Stop() {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	pr.running = false
	pr.current = 0
	pr.total = 0
	pr.video = ""
}

// GetProgress returns the current progress (thread-safe)
func (pr *PreviewRegenerator) GetProgress() RegenerationProgress {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return RegenerationProgress{
		Current:      pr.current,
		Total:        pr.total,
		CurrentVideo: pr.video,
		IsRunning:    pr.running,
	}
}

// IsRunning returns whether regeneration is currently running (thread-safe)
func (pr *PreviewRegenerator) IsRunning() bool {
	pr.mu.RLock()
	defer pr.mu.RUnlock()
	return pr.running
}

var previewRegenerator = NewPreviewRegenerator()

// RegenerateAllPreviews deletes and regenerates preview frames for all recordings
func RegenerateAllPreviews() error {
	if previewRegenerator.IsRunning() {
		return fmt.Errorf("preview regeneration is already running")
	}

	go func() {
		log.Infoln("[RegenerateAllPreviews] Starting preview regeneration for all recordings")

		recordings, err := db.RecordingsList()
		if err != nil {
			log.Errorf("[RegenerateAllPreviews] Error fetching recordings: %v", err)
			ws.BroadCastClients(ws.JobErrorEvent, fmt.Sprintf("Failed to fetch recordings: %v", err))
			return
		}

		total := len(recordings)
		previewRegenerator.Start(total)

		ws.BroadCastClients(ws.JobStartEvent, map[string]interface{}{
			"type":    "preview_regeneration",
			"message": fmt.Sprintf("Starting preview regeneration for %d videos", total),
			"total":   total,
		})

		successCount := 0
		errorCount := 0

		for i, rec := range recordings {
			current := i + 1
			videoName := fmt.Sprintf("%s/%s", rec.ChannelName, rec.Filename)
			previewRegenerator.Update(current, videoName)

			log.Infof("[RegenerateAllPreviews] Processing %s (%d/%d)", videoName, current, total)

			// Broadcast progress update
			ws.BroadCastClients(ws.JobProgressEvent, map[string]interface{}{
				"type":         "preview_regeneration",
				"current":      current,
				"total":        total,
				"currentVideo": videoName,
			})

			// Delete existing preview frames
			previewPath := rec.RecordingID.GetPreviewFramesPath(rec.ChannelName)
			if err := os.RemoveAll(previewPath); err != nil && !os.IsNotExist(err) {
				log.Errorf("[RegenerateAllPreviews] Error removing existing previews for %s: %v", videoName, err)
				errorCount++
				continue
			}

			// Create job to regenerate preview frames
			_, err = rec.EnqueuePreviewFramesJob()
			if err != nil {
				log.Errorf("[RegenerateAllPreviews] Error creating preview job for %s: %v", videoName, err)
				errorCount++
				continue
			}

			successCount++
			log.Infof("[RegenerateAllPreviews] Created preview job for %s (%d/%d)", videoName, current, total)
		}

		previewRegenerator.Stop()

		log.Infof("[RegenerateAllPreviews] Completed: %d successful, %d errors out of %d total", successCount, errorCount, total)

		ws.BroadCastClients(ws.JobDoneEvent, map[string]interface{}{
			"type":       "preview_regeneration",
			"message":    fmt.Sprintf("Preview regeneration completed: %d successful, %d errors", successCount, errorCount),
			"successful": successCount,
			"errors":     errorCount,
			"total":      total,
		})
	}()

	return nil
}

// GetRegenerationProgress returns the current regeneration progress
func GetRegenerationProgress() RegenerationProgress {
	return previewRegenerator.GetProgress()
}

// IsRegeneratingPreviews returns whether preview regeneration is currently running
func IsRegeneratingPreviews() bool {
	return previewRegenerator.IsRunning()
}
