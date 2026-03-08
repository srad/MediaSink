package handlers

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/util"
	"github.com/srad/mediasink/internal/ws"
)

// EmitJobProgress reports job progress to database and connected clients
func EmitJobProgress(job *db.Job, current, total uint64, message string) {
	// Calculate percentage for database storage
	progressPercent := float32(current) / float32(total) * 100

	// Update database
	if err := job.UpdateProgress(fmt.Sprintf("%.2f", progressPercent)); err != nil {
		log.Errorf("Error updating job progress in database: %s", err)
	}

	// Broadcast to connected clients
	ws.BroadCastClients(ws.JobProgressEvent, JobMessage[util.TaskProgress]{
		Job: job,
		Data: util.TaskProgress{
			Step:    1,
			Steps:   1,
			Total:   total,
			Current: current,
			Message: message,
		},
	})
}

// EmitProgressFromFrame parses FFmpeg frame output and emits progress
func EmitProgressFromFrame(job *db.Job, s string, totalCount uint64) {
	if strings.Contains(s, "frame=") {
		s := strings.Split(s, "=")
		if len(s) == 2 {
			if p, err := strconv.ParseUint(s[1], 10, 64); err == nil {
				EmitJobProgress(job, p, totalCount, "Processing")
			}
		}
	}
}
