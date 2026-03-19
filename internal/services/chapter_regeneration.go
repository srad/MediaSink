package services

import (
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/models/responses"
	"github.com/srad/mediasink/internal/ws"
)

// RegenerateAllChapters removes all existing chapter-analysis jobs and enqueues
// the next valid analysis step for every recording in the library.
// Recordings with missing/invalid previews get a preview job first.
func RegenerateAllChapters() (*responses.RegenerateChaptersResponse, error) {
	removedIDs, err := db.PurgeJobsByTask(db.TaskAnalyzeFrames)
	if err != nil {
		return nil, err
	}

	for _, id := range removedIDs {
		ws.BroadCastClients(ws.JobDeleteEvent, id)
	}

	recordings, err := db.RecordingsList()
	if err != nil {
		return nil, err
	}

	enqueued := 0
	for _, recording := range recordings {
		if recording == nil {
			continue
		}
		if _, err := EnqueueAnalysisPipeline(recording); err != nil {
			log.Warnf("[RegenerateAllChapters] Failed to enqueue next analysis step for recording %d: %v", recording.RecordingID, err)
			continue
		}
		enqueued++
	}

	return &responses.RegenerateChaptersResponse{
		RemovedJobs: len(removedIDs),
		Enqueued:    enqueued,
		Recordings:  len(recordings),
	}, nil
}
