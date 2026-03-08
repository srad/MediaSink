package handlers

import (
	"fmt"
	"os"

	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/util"
	"github.com/srad/mediasink/internal/ws"
)

// conversionHandler converts videos to different formats
type conversionHandler struct {
	deps *HandlerDependencies
}

var _ JobHandler = (*conversionHandler)(nil)

// NewConversionHandler creates a new conversion handler
func NewConversionHandler(deps *HandlerDependencies) JobHandler {
	return &conversionHandler{
		deps: deps,
	}
}

// Name returns the handler name
func (h *conversionHandler) Name() string {
	return "conversion"
}

// Handle converts a video to a different format
func (h *conversionHandler) Handle(job *db.Job, threadCount int) error {
	mediaType, err := db.UnmarshalJobArg[string](job)
	if err != nil {
		return err
	}

	result, errConvert := util.ConvertVideo(&util.VideoConversionArgs{
		OnStart: func(info util.TaskInfo) {
			if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
				h.deps.Logger.Errorf("Error updating job info: %s", err)
			}
		},
		OnProgress: func(info util.TaskProgress) {
			EmitJobProgress(job, info.Current, info.Total, info.Message)
		},
		OnError: func(err error) {
			ws.BroadCastClients(ws.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
		},
		InputPath:   job.ChannelName.AbsoluteChannelPath(),
		Filename:    job.Filename.String(),
		OutputPath:  job.ChannelName.AbsoluteChannelPath(),
		ThreadCount: threadCount,
	}, *mediaType)

	if errConvert != nil {
		message := fmt.Errorf("error converting %s to %s: %w", job.Filename, *mediaType, errConvert)

		h.deps.Logger.Error(message)
		if errDelete := os.Remove(result.Filepath); errDelete != nil {
			h.deps.Logger.Errorf("error deleting file %s: %v", result.Filepath, errDelete)
		}
		return message
	}

	h.deps.Logger.Infof("[conversionJobs] Completed conversion of '%s' with args '%s'", job.Filename, *job.Args)

	// Create recording entry for converted file and enqueue previews job
	recording, err := db.CreateRecording(job.ChannelID, db.RecordingFileName(result.Filename), "recording")
	if err != nil {
		// Failed to create recording, clean up the converted file
		if errRemove := os.Remove(result.Filepath); errRemove != nil {
			return fmt.Errorf("error creating recording: %w (also failed to delete file: %w)", err, errRemove)
		}
		return fmt.Errorf("error creating recording: %w", err)
	}

	// Enqueue preview generation job for the newly created recording
	if _, errPreviews := recording.EnqueuePreviewFramesJob(); errPreviews != nil {
		// Preview job enqueue failed - clean up the converted file and database record
		h.deps.Logger.Errorf("Error enqueueing preview job, cleaning up converted file: %v (will attempt cleanup)", errPreviews)

		// Try to delete the recording from database
		if errDelete := recording.DestroyRecording(); errDelete != nil {
			h.deps.Logger.Errorf("Error deleting orphaned recording from database: %v", errDelete)
			// Return both errors so caller knows about the cleanup failure
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreviews, errDelete)
		}

		return fmt.Errorf("error enqueueing preview job for recording: %w", errPreviews)
	}

	h.deps.Logger.Infof("Conversion completed for %s", job.Filepath)

	return nil
}
