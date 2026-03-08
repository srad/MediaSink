package jobs

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/jobs/handlers"
	"github.com/srad/mediasink/internal/ws"
)

// analyzeFrameHandler is set by services/video_analysis.go to handle frame analysis
// This avoids a circular import between jobs and services packages
var analyzeFrameHandler func(*db.Job) error

// RegisterAnalyzeFrameHandler registers the frame analysis handler
// Must be called during initialization by services/video_analysis.go
func RegisterAnalyzeFrameHandler(handler func(*db.Job) error) {
	analyzeFrameHandler = handler
}

// executeJob dispatches the job to the appropriate handler based on task type
func executeJob(job *db.Job, threadCount int) error {
	// Create handler dependencies
	deps := handlers.NewHandlerDependencies(log.StandardLogger(), db.DB)

	// Create handler for this job type and execute it
	var jobHandler handlers.JobHandler

	switch job.Task {
	case db.TaskPreviewFrames:
		jobHandler = handlers.NewPreviewFramesHandler(deps)
	case db.TaskAnalyzeFrames:
		if analyzeFrameHandler == nil {
			return handleJob(job, fmt.Errorf("frame analysis handler not registered"))
		}
		return handleJob(job, analyzeFrameHandler(job))
	case db.TaskCut:
		jobHandler = handlers.NewCuttingHandler(deps)
	case db.TaskConvert:
		jobHandler = handlers.NewConversionHandler(deps)
	case db.TaskMerge:
		jobHandler = handlers.NewMergeHandler(deps)
	case db.TaskEnhanceVideo:
		jobHandler = handlers.NewEnhanceHandler(deps)
	default:
		return nil
	}

	// Execute the job handler
	return handleJob(job, jobHandler.Handle(job, threadCount))
}

// handleJob processes the result of a job execution and updates the job state
func handleJob(job *db.Job, err error) error {
	if err != nil {
		errErrStore := job.Error(err)
		ws.BroadCastClients(ws.JobErrorEvent, JobMessage[string]{Data: err.Error(), Job: job})
		return errErrStore
	} else {
		return job.Completed()
	}
}
