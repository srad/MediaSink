package handlers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"

	"github.com/srad/mediasink/network"
)

// mergeHandler merges multiple video recordings into a single file
type mergeHandler struct {
	deps *HandlerDependencies
}

var _ JobHandler = (*mergeHandler)(nil)

// NewMergeHandler creates a new merge handler
func NewMergeHandler(deps *HandlerDependencies) JobHandler {
	return &mergeHandler{
		deps: deps,
	}
}

// Name returns the handler name
func (h *mergeHandler) Name() string {
	return "merge"
}

// Handle merges multiple video recordings into a single file
func (h *mergeHandler) Handle(job *database.Job, threadCount int) error {
	mergeArgs, err := database.UnmarshalJobArg[helpers.MergeJobArgs](job)
	if err != nil {
		return err
	}

	recordingIDs := mergeArgs.RecordingIDs
	reEncode := mergeArgs.ReEncode

	// Collect all recording file paths
	inputFiles := make([]string, len(recordingIDs))
	for i, recID := range recordingIDs {
		rec, err := database.RecordingID(recID).FindRecordingByID()
		if err != nil {
			return fmt.Errorf("failed to find recording %d: %w", recID, err)
		}
		inputFiles[i] = rec.AbsoluteChannelFilepath()
	}

	// Generate output filename
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	filename := database.RecordingFileName(fmt.Sprintf("%s_merged_%s.mp4", job.ChannelName, stamp))
	outputFile := job.ChannelName.AbsoluteChannelFilePath(filename)

	h.deps.Logger.Infof("[MergeJob] Starting merge of %d recordings", len(recordingIDs))

	var errMerge error
	if reEncode {
		// Re-encoded merge
		errMerge = helpers.MergeVideosReEncoded(&helpers.MergeReEncodeArgs{
			OnStart: func(info helpers.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					h.deps.Logger.Errorf("[MergeJob] Error updating job info: %v", err)
				}
				network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
					Job: job,
					Data: helpers.TaskInfo{
						Pid:     info.Pid,
						Command: info.Command,
						Message: "Starting re-encoded merge",
					},
				})
			},
			OnProgress: func(info helpers.PipeMessage) {
				network.BroadCastClients(network.JobProgressEvent, JobMessage[string]{Job: job, Data: info.Output})
			},
			OnErr: func(err error) {
				network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
			},
			InputFiles:             inputFiles,
			AbsoluteOutputFilepath: outputFile,
		})
	} else {
		// Regular concat merge (requires same properties)
		concatFileContent := make([]string, len(inputFiles))
		for i, file := range inputFiles {
			concatFileContent[i] = fmt.Sprintf("file '%s'", file)
		}

		concatFilePath := job.ChannelName.AbsoluteChannelFilePath(database.RecordingFileName(fmt.Sprintf("merge_concat_%d.txt", now.UnixNano())))
		errWrite := os.WriteFile(concatFilePath, []byte(strings.Join(concatFileContent, "\n")), 0644)
		if errWrite != nil {
			h.deps.Logger.Errorf("[MergeJob] Error writing concat file: %v", errWrite)
			return fmt.Errorf("error writing concat file: %w", errWrite)
		}

		errMerge = helpers.MergeVideos(&helpers.MergeArgs{
			OnStart: func(info helpers.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					h.deps.Logger.Errorf("[MergeJob] Error updating job info: %v", err)
				}
				network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
					Job: job,
					Data: helpers.TaskInfo{
						Pid:     info.Pid,
						Command: info.Command,
						Message: "Starting merge",
					},
				})
			},
			OnProgress: func(info helpers.PipeMessage) {
				network.BroadCastClients(network.JobProgressEvent, JobMessage[string]{Job: job, Data: info.Output})
			},
			OnErr: func(err error) {
				network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
			},
			MergeFileAbsolutePath:  concatFilePath,
			AbsoluteOutputFilepath: outputFile,
		})

		// Clean up concat file
		if errCleanup := os.Remove(concatFilePath); errCleanup != nil {
			h.deps.Logger.Warnf("[MergeJob] Error deleting concat file: %v", errCleanup)
		}
	}

	if errMerge != nil {
		h.deps.Logger.Errorf("[MergeJob] Error merging recordings: %v", errMerge)
		return errMerge
	}

	// Verify merged output
	outputVideo := &helpers.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("[MergeJob] Error deleting unreadable merged file: %v", errCleanup)
		}
		return fmt.Errorf("error validating merged video: %w", err)
	}

	// Create recording entry in database
	mergedRecording, errCreate := database.CreateRecording(job.ChannelID, filename, "merged")
	if errCreate != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("[MergeJob] Error deleting merged file after DB save failure: %v", errCleanup)
		}
		return fmt.Errorf("error creating merged recording in database: %w", errCreate)
	}

	// Enqueue preview jobs for merged recording
	if _, errPreview := mergedRecording.EnqueuePreviewFramesJob(); errPreview != nil {
		h.deps.Logger.Errorf("[MergeJob] Error enqueueing preview jobs for merged recording: %v", errPreview)

		if errDelete := mergedRecording.DestroyRecording(); errDelete != nil {
			h.deps.Logger.Errorf("[MergeJob] Error cleaning up orphaned merged recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return fmt.Errorf("error enqueueing preview job: %w", errPreview)
	}

	h.deps.Logger.Infof("[MergeJob] Successfully merged %d recordings into %s", len(recordingIDs), filename)
	return nil
}
