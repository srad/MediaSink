package handlers

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/util"

	"github.com/srad/mediasink/internal/ws"
)

// cuttingHandler handles the three-phase cutting job:
// 1. Cut video at the given time intervals
// 2. Merge the cuts
// 3. Enqueue preview job for new cut
type cuttingHandler struct {
	deps *HandlerDependencies
}

var _ JobHandler = (*cuttingHandler)(nil)

// NewCuttingHandler creates a new cutting handler
func NewCuttingHandler(deps *HandlerDependencies) JobHandler {
	return &cuttingHandler{
		deps: deps,
	}
}

// Name returns the handler name
func (h *cuttingHandler) Name() string {
	return "cutting"
}

// Handle processes a video cutting job
func (h *cuttingHandler) Handle(job *db.Job, threadCount int) error {
	cutArgs, err := db.UnmarshalJobArg[util.CutArgs](job)
	if err != nil {
		return err
	}

	// Validate cut arguments
	if len(cutArgs.Starts) != len(cutArgs.Ends) {
		return fmt.Errorf("cut arguments validation failed: number of starts (%d) does not match number of ends (%d)", len(cutArgs.Starts), len(cutArgs.Ends))
	}

	for i := range cutArgs.Starts {
		if cutArgs.Starts[i] >= cutArgs.Ends[i] {
			return fmt.Errorf("cut arguments validation failed: start time (%s) must be before end time (%s) for segment %d", cutArgs.Starts[i], cutArgs.Ends[i], i)
		}
	}

	h.deps.Logger.Infof("[Job] Generating video cut for '%s'", job.Filename)

	// Filenames
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	filename := db.RecordingFileName(fmt.Sprintf("%s_cut_%s.mp4", job.ChannelName, stamp))
	inputPath := job.ChannelName.AbsoluteChannelFilePath(job.Filename)
	outputFile := job.ChannelName.AbsoluteChannelFilePath(filename)
	segFiles := make([]string, len(cutArgs.Starts))
	mergeFileContent := make([]string, len(cutArgs.Starts))

	// Cut
	segmentFilename := fmt.Sprintf("%s_cut_%s", job.ChannelName, stamp)
	for i, start := range cutArgs.Starts {
		segFiles[i] = job.ChannelName.AbsoluteChannelFilePath(db.RecordingFileName(fmt.Sprintf("%s_%04d.mp4", segmentFilename, i)))
		err = util.CutVideo(&util.CuttingJob{
			OnStart: func(info *util.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					h.deps.Logger.Errorf("[Job] Error updating job info during cut: %v", err)
				}

				ws.BroadCastClients(ws.JobStartEvent, JobMessage[util.TaskInfo]{
					Job: job,
					Data: util.TaskInfo{
						Steps:   2,
						Step:    1,
						Pid:     info.Pid,
						Command: info.Command,
						Message: "Starting cutting phase",
					},
				})
			},
			OnProgress: func(s string) {
				ws.BroadCastClients(ws.JobProgressEvent, JobMessage[string]{Job: job, Data: s})
			},
		}, inputPath, segFiles[i], start, cutArgs.Ends[i])
		// Failed, delete all segments
		if err != nil {
			h.deps.Logger.Errorf("[Job] Error generating cut for file '%s': %s", inputPath, err)
			h.deps.Logger.Infoln("[Job] Deleting orphaned segments")
			for _, file := range segFiles {
				if err := os.RemoveAll(file); err != nil {
					h.deps.Logger.Errorf("[Job] Error deleting segment '%s': %s", file, err)
				}
			}
			return err
		}
	}
	// Merge file txt, enumerate
	for i, file := range segFiles {
		mergeFileContent[i] = fmt.Sprintf("file '%s'", file)
	}
	mergeFileAbsolutePath := job.ChannelName.AbsoluteChannelFilePath(db.RecordingFileName(fmt.Sprintf("%s.txt", segmentFilename)))
	errWriteMergeFile := os.WriteFile(mergeFileAbsolutePath, []byte(strings.Join(mergeFileContent, "\n")), 0644)
	if errWriteMergeFile != nil {
		h.deps.Logger.Errorf("[Job] Error writing concat text file %s: %s", mergeFileAbsolutePath, errWriteMergeFile)
		for _, file := range segFiles {
			if err := os.RemoveAll(file); err != nil {
				h.deps.Logger.Errorf("[Job] Error deleting %s: %s", file, err)
			}
		}
		if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
			h.deps.Logger.Errorf("[Job] Error deleting merge file %s: %s", mergeFileAbsolutePath, err)
		}
		return errWriteMergeFile
	}

	errMerge := util.MergeVideos(&util.MergeArgs{
		OnStart: func(info util.CommandInfo) {
			ws.BroadCastClients(ws.JobStartEvent, JobMessage[util.TaskInfo]{
				Job: job,
				Data: util.TaskInfo{
					Steps:   2,
					Step:    2,
					Pid:     info.Pid,
					Command: info.Command,
					Message: "Starting merge phase",
				},
			})
		},
		OnProgress: func(info util.PipeMessage) {
			// TODO: For cutting and merging ffmpeg doesnt seem to provide obvious progress information, check again.
			//ws.BroadCastClients("job:progress", JobMessage{Job: job, Data: info})
		},
		OnErr: func(err error) {
			ws.BroadCastClients(ws.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
		},
		MergeFileAbsolutePath:  mergeFileAbsolutePath,
		AbsoluteOutputFilepath: outputFile,
	})

	if errMerge != nil {
		// Job failed, destroy all files.
		h.deps.Logger.Errorf("Error merging file '%s': %s", mergeFileAbsolutePath, errMerge)
		for _, file := range segFiles {
			if err := os.RemoveAll(file); err != nil {
				h.deps.Logger.Errorf("Error deleting %s: %s", file, err)
			}
		}
		if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
			h.deps.Logger.Errorf("Error deleting merge file %s: %s", mergeFileAbsolutePath, err)
		}
		return errMerge
	}

	// Clean up merge file
	if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
		h.deps.Logger.Warnf("Error deleting merge file %s: %v", mergeFileAbsolutePath, err)
	}

	// Clean up segment files
	for _, file := range segFiles {
		h.deps.Logger.Infof("[MergeJob] Deleting segment %s", file)
		if err := os.Remove(file); err != nil {
			h.deps.Logger.Errorf("Error deleting segment '%s': %v", file, err)
		}
	}

	// Verify output video was created and readable
	outputVideo := &util.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		// Merged file is unreadable, clean up before returning
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("Error deleting unreadable merged file '%s': %v", outputFile, errCleanup)
		}
		return fmt.Errorf("error reading video information for merged file '%s': %w", filename, err)
	}

	cutRecording, errCreate := db.CreateRecording(job.ChannelID, filename, "cut")
	if errCreate != nil {
		// Failed to create recording - clean up the merged file
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("Error deleting merged file '%s' after DB save failure: %v", outputFile, errCleanup)
		}
		return errCreate
	}

	// Successfully added cut record, enqueue preview job
	if _, errPreview := cutRecording.EnqueuePreviewFramesJob(); errPreview != nil {
		// Preview job enqueue failed - clean up the cut recording and file
		h.deps.Logger.Errorf("Error enqueueing preview job for cut recording, cleaning up: %v", errPreview)

		if errDelete := cutRecording.DestroyRecording(); errDelete != nil {
			h.deps.Logger.Errorf("Error cleaning up orphaned cut recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return errPreview
	}

	// The original file shall be deleted after the process if successful.
	if cutArgs.DeleteAfterCompletion {
		recording, err := db.FindRecordingByID(job.RecordingID)
		if err != nil {
			return err
		}
		return recording.DestroyRecording()
	}

	return nil
}
