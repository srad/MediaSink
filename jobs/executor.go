package jobs

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/jobs/handlers"
	"github.com/srad/mediasink/network"
)

// analyzeFrameHandler is set by services/video_analysis.go to handle frame analysis
// This avoids a circular import between jobs and services packages
var analyzeFrameHandler func(*database.Job) error

// RegisterAnalyzeFrameHandler registers the frame analysis handler
// Must be called during initialization by services/video_analysis.go
func RegisterAnalyzeFrameHandler(handler func(*database.Job) error) {
	analyzeFrameHandler = handler
}

// executeJob dispatches the job to the appropriate handler based on task type
func executeJob(job *database.Job, threadCount int) error {
	// Create handler dependencies
	deps := handlers.NewHandlerDependencies(log.StandardLogger(), database.DB)

	// Create handler for this job type and execute it
	var jobHandler handlers.JobHandler

	switch job.Task {
	case database.TaskPreviewFrames:
		jobHandler = handlers.NewPreviewFramesHandler(deps)
	case database.TaskAnalyzeFrames:
		if analyzeFrameHandler == nil {
			return handleJob(job, fmt.Errorf("frame analysis handler not registered"))
		}
		return handleJob(job, analyzeFrameHandler(job))
	case database.TaskCut:
		jobHandler = handlers.NewCuttingHandler(deps)
	case database.TaskConvert:
		jobHandler = handlers.NewConversionHandler(deps)
	case database.TaskMerge:
		jobHandler = handlers.NewMergeHandler(deps)
	case database.TaskEnhanceVideo:
		jobHandler = handlers.NewEnhanceHandler(deps)
	default:
		return nil
	}

	// Execute the job handler
	return handleJob(job, jobHandler.Handle(job, threadCount))
}

// handleJob processes the result of a job execution and updates the job state
func handleJob(job *database.Job, err error) error {
	if err != nil {
		errErrStore := job.Error(err)
		network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Data: err.Error(), Job: job})
		return errErrStore
	} else {
		return job.Completed()
	}
}
