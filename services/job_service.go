package services

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/srad/mediasink/conf"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"
	"github.com/srad/mediasink/network"
)

var (
	sleepBetweenRounds  = 1 * time.Second
	ctxJobs, cancelJobs = context.WithCancel(context.Background())
	processing          = false
	processingMutex     sync.Mutex
)

type JobMessage[T any] struct {
	Job  *database.Job `json:"job"`
	Data T             `json:"data"`
}

func processJobs(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Infoln("[processJobs] Worker stopped")
			processingMutex.Lock()
			processing = false
			processingMutex.Unlock()
			return
		case <-time.After(sleepBetweenRounds):
			processingMutex.Lock()
			processing = true
			processingMutex.Unlock()
			job, errNextJob := database.GetNextJob()
			if errNextJob != nil {
				if job != nil {
					_ = job.Error(errNextJob)
				} else {
					log.Errorf("Failed to get next job: %s", errNextJob)
				}
				continue
			}
			if job == nil {
				continue
			}

			if err := job.Activate(); err != nil {
				// Activation failed - mark job as error and skip to next iteration
				log.Errorf("Error activating job %d: %v", job.JobID, err)
				if errMark := job.Error(fmt.Errorf("failed to activate job: %w", err)); errMark != nil {
					log.Errorf("Error marking job %d as failed: %v", job.JobID, errMark)
				}
				network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{
					Job: job,
					Data: fmt.Sprintf("Failed to activate job: %v", err),
				})
				continue
			}
			network.BroadCastClients(network.JobActivate, JobMessage[any]{Job: job})

			// Execute the job (this will update job state: Completed or Error)
			if err := executeJob(job); err != nil {
				log.Errorf("Job execution error: %v", err)
				// Job state already updated by executeJob -> handleJob -> job.Error()
			}

			// Deactivate the job - it should already be inactive after job.Complete() or job.Error()
			// but we ensure it's deactivated in case of state inconsistencies
			if err := job.Deactivate(); err != nil {
				log.Warnf("Error deactivating job %d after execution: %v - job may remain in active state", job.JobID, err)
			}

			// Broadcast final state after deactivation
			network.BroadCastClients(network.JobDeactivate, JobMessage[any]{Job: job})
		}
	}
}

// executeJob Blocking execution.
func executeJob(job *database.Job) error {
	video := helpers.Video{FilePath: job.Recording.AbsoluteChannelFilepath()}

	switch job.Task {
	case database.TaskPreviewCover:
		return handleJob(job, processPreviewCover(job, &video))
	case database.TaskPreviewStrip:
		return handleJob(job, processPreviewStrip(job, &video))
	case database.TaskPreviewVideo:
		// video jobs won't be created for now.
		return nil //handleJob(job, processPreviewVideo(job, &video))
	case database.TaskCut:
		return handleJob(job, processCutting(job))
	case database.TaskConvert:
		return handleJob(job, processConversion(job))
	}

	return nil
}

func handleJob(job *database.Job, err error) error {
	if err != nil {
		errErrStore := job.Error(err)
		network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Data: err.Error(), Job: job})
		return errErrStore
	} else {
		return job.Completed()
	}
}

func processPreviewStrip(job *database.Job, video *helpers.Video) error {
	previewArgs := &helpers.VideoConversionArgs{
		OnStart: func(info helpers.TaskInfo) {
			if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
				log.Errorf("[Job] Error updating job info: %s", err)
			}

			network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
				Job:  job,
				Data: info,
			})
		},
		OnProgress: func(info helpers.TaskProgress) {
			network.BroadCastClients(network.JobProgressEvent, JobMessage[helpers.TaskProgress]{
				Job:  job,
				Data: info})
		},
		OnEnd: func(info helpers.TaskComplete) {
			network.BroadCastClients(network.JobDoneEvent, JobMessage[helpers.TaskComplete]{
				Data: info,
				Job:  job,
			})
		},
		OnError: func(err error) {
			network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{
				Data: err.Error(),
				Job:  job,
			})
		},
		InputPath:  job.ChannelName.AbsoluteChannelPath(),
		OutputPath: job.ChannelName.AbsoluteChannelDataPath(),
		Filename:   job.Filename.String(),
	}

	if _, err := video.ExecPreviewStripe(previewArgs, conf.FrameCount, 256, job.Recording.Packets); err != nil {
		return err
	} else {
		return job.Recording.UpdatePreviewPath(database.PreviewStripe)
	}
}

func processPreviewVideo(job *database.Job, video *helpers.Video) error {
	if err := job.Recording.DestroyPreview(database.PreviewVideo); err != nil {
		return err
	}

	previewArgs := &helpers.VideoConversionArgs{
		OnStart: func(info helpers.TaskInfo) {
			if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
				log.Errorf("[Job] Error updating job info: %s", err)
			}

			network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
				Job:  job,
				Data: info,
			})
		},
		OnProgress: func(info helpers.TaskProgress) {
			network.BroadCastClients(network.JobProgressEvent, JobMessage[helpers.TaskProgress]{
				Job:  job,
				Data: info})
		},
		OnEnd: func(info helpers.TaskComplete) {
			network.BroadCastClients(network.JobDoneEvent, JobMessage[helpers.TaskComplete]{
				Data: info,
				Job:  job,
			})
		},
		OnError: func(err error) {
			network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{
				Data: err.Error(),
				Job:  job,
			})
		},
		InputPath:  job.ChannelName.AbsoluteChannelPath(),
		OutputPath: job.ChannelName.AbsoluteChannelDataPath(),
		Filename:   job.Filename.String(),
	}

	if _, err := video.ExecPreviewVideo(previewArgs, conf.FrameCount, 256, job.Recording.Packets); err != nil {
		return err
	}

	return job.Recording.UpdatePreviewPath(database.PreviewVideo)
}

func processPreviewCover(job *database.Job, video *helpers.Video) error {
	if _, err := video.ExecPreviewCover(job.ChannelName.AbsoluteChannelDataPath()); err != nil {
		return err
	}
	return job.Recording.UpdatePreviewPath(database.PreviewCover)
}

func processConversion(job *database.Job) error {
	mediaType, err := database.UnmarshalJobArg[string](job)
	if err != nil {
		return err
	}

	result, errConvert := helpers.ConvertVideo(&helpers.VideoConversionArgs{
		OnStart: func(info helpers.TaskInfo) {
			if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
				log.Errorf("Error updating job info: %s", err)
			}
		},
		OnProgress: func(info helpers.TaskProgress) {
			if err := job.UpdateProgress(fmt.Sprintf("%f", float32(info.Current)/float32(info.Total)*100)); err != nil {
				log.Errorf("Error updating job progress: %s", err)
			}

			network.BroadCastClients(network.JobProgressEvent, JobMessage[helpers.TaskProgress]{Job: job, Data: info})
		},
		OnError: func(err error) {
			network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
		},
		InputPath:  job.ChannelName.AbsoluteChannelPath(),
		Filename:   job.Filename.String(),
		OutputPath: job.ChannelName.AbsoluteChannelPath(),
	}, *mediaType)

	if errConvert != nil {
		message := fmt.Errorf("error converting %s to %s: %w", job.Filename, *mediaType, errConvert)

		log.Errorln(message)
		if errDelete := os.Remove(result.Filepath); errDelete != nil {
			log.Errorf("error deleting file %s: %w", result.Filepath, errDelete)
		}
		return message
	}

	log.Infof("[conversionJobs] Completed conversion of '%s' with args '%s'", job.Filename, *job.Args)

	// Create recording entry for converted file and enqueue previews job
	recording, err := database.CreateRecording(job.ChannelID, database.RecordingFileName(result.Filename), "recording")
	if err != nil {
		// Failed to create recording, clean up the converted file
		if errRemove := os.Remove(result.Filepath); errRemove != nil {
			return fmt.Errorf("error creating recording: %w (also failed to delete file: %w)", err, errRemove)
		}
		return fmt.Errorf("error creating recording: %w", err)
	}

	// Enqueue preview generation job for the newly created recording
	if _, _, errPreviews := recording.EnqueuePreviewsJob(); errPreviews != nil {
		// Preview job enqueue failed - clean up the converted file and database record
		log.Errorf("Error enqueueing preview job, cleaning up converted file: %v (will attempt cleanup)", errPreviews)

		// Try to delete the recording from database
		if errDelete := recording.DestroyRecording(); errDelete != nil {
			log.Errorf("Error deleting orphaned recording from database: %v", errDelete)
			// Return both errors so caller knows about the cleanup failure
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreviews, errDelete)
		}

		return fmt.Errorf("error enqueueing preview job for recording: %w", errPreviews)
	}

	log.Infof("Conversion completed for %s", job.Filepath)

	return nil
}

// Three-phase cutting job:
// 1. Cut video at the given time intervals
// 2. Merge the cuts
// 3. Enqueue preview job for new cut
// This action is intrinsically procedural, keep it together locally.
func processCutting(job *database.Job) error {
	cutArgs, err := database.UnmarshalJobArg[helpers.CutArgs](job)
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

	log.Infof("[Job] Generating video cut for '%s'", job.Filename)

	// Filenames
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	filename := database.RecordingFileName(fmt.Sprintf("%s_cut_%s.mp4", job.ChannelName, stamp))
	inputPath := job.ChannelName.AbsoluteChannelFilePath(job.Filename)
	outputFile := job.ChannelName.AbsoluteChannelFilePath(filename)
	segFiles := make([]string, len(cutArgs.Starts))
	mergeFileContent := make([]string, len(cutArgs.Starts))

	// Cut
	segmentFilename := fmt.Sprintf("%s_cut_%s", job.ChannelName, stamp)
	for i, start := range cutArgs.Starts {
		segFiles[i] = job.ChannelName.AbsoluteChannelFilePath(database.RecordingFileName(fmt.Sprintf("%s_%04d.mp4", segmentFilename, i)))
		err = helpers.CutVideo(&helpers.CuttingJob{
			OnStart: func(info *helpers.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					log.Errorf("[Job] Error updating job info during cut: %v", err)
				}

				network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
					Job: job,
					Data: helpers.TaskInfo{
						Steps:   2,
						Step:    1,
						Pid:     info.Pid,
						Command: info.Command,
						Message: "Starting cutting phase",
					},
				})
			},
			OnProgress: func(s string) {
				network.BroadCastClients(network.JobProgressEvent, JobMessage[string]{Job: job, Data: s})
			},
		}, inputPath, segFiles[i], start, cutArgs.Ends[i])
		// Failed, delete all segments
		if err != nil {
			log.Errorf("[Job] Error generating cut for file '%s': %s", inputPath, err)
			log.Infoln("[Job] Deleting orphaned segments")
			for _, file := range segFiles {
				if err := os.RemoveAll(file); err != nil {
					log.Errorf("[Job] Error deleting segment '%s': %s", file, err)
				}
			}
			return err
		}
	}
	// Merge file txt, enumerate
	for i, file := range segFiles {
		mergeFileContent[i] = fmt.Sprintf("file '%s'", file)
	}
	mergeFileAbsolutePath := job.ChannelName.AbsoluteChannelFilePath(database.RecordingFileName(fmt.Sprintf("%s.txt", segmentFilename)))
	errWriteMergeFile := os.WriteFile(mergeFileAbsolutePath, []byte(strings.Join(mergeFileContent, "\n")), 0644)
	if errWriteMergeFile != nil {
		log.Errorf("[Job] Error writing concat text file %s: %s", mergeFileAbsolutePath, errWriteMergeFile)
		for _, file := range segFiles {
			if err := os.RemoveAll(file); err != nil {
				log.Errorf("[Job] Error deleting %s: %s", file, err)
			}
		}
		if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
			log.Errorf("[Job] Error deleting merge file %s: %s", mergeFileAbsolutePath, err)
		}
		return errWriteMergeFile
	}

	errMerge := helpers.MergeVideos(&helpers.MergeArgs{
		OnStart: func(info helpers.CommandInfo) {
			network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
				Job: job,
				Data: helpers.TaskInfo{
					Steps:   2,
					Step:    2,
					Pid:     info.Pid,
					Command: info.Command,
					Message: "Starting merge phase",
				},
			})
		},
		OnProgress: func(info helpers.PipeMessage) {
			// TODO: For cutting and merging ffmpeg doesnt seem to provide obvious progress information, check again.
			//network.BroadCastClients("job:progress", JobMessage{Job: job, Data: info})
		},
		OnErr: func(err error) {
			network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: err.Error()})
		},
		MergeFileAbsolutePath:  mergeFileAbsolutePath,
		AbsoluteOutputFilepath: outputFile,
	})

	if errMerge != nil {
		// Job failed, destroy all files.
		log.Errorf("Error merging file '%s': %s", mergeFileAbsolutePath, errMerge)
		for _, file := range segFiles {
			if err := os.RemoveAll(file); err != nil {
				log.Errorf("Error deleting %s: %s", file, err)
			}
		}
		if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
			log.Errorf("Error deleting merge file %s: %s", mergeFileAbsolutePath, err)
		}
		return errMerge
	}

	// Clean up merge file
	if err := os.RemoveAll(mergeFileAbsolutePath); err != nil {
		log.Warnf("Error deleting merge file %s: %v", mergeFileAbsolutePath, err)
	}

	// Clean up segment files
	for _, file := range segFiles {
		log.Infof("[MergeJob] Deleting segment %s", file)
		if err := os.Remove(file); err != nil {
			log.Errorf("Error deleting segment '%s': %v", file, err)
		}
	}

	// Verify output video was created and readable
	outputVideo := &helpers.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		// Merged file is unreadable, clean up before returning
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("Error deleting unreadable merged file '%s': %v", outputFile, errCleanup)
		}
		return fmt.Errorf("error reading video information for merged file '%s': %w", filename, err)
	}

	cutRecording, errCreate := database.CreateRecording(job.ChannelID, filename, "cut")
	if errCreate != nil {
		// Failed to create recording - clean up the merged file
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("Error deleting merged file '%s' after DB save failure: %v", outputFile, errCleanup)
		}
		return errCreate
	}

	// Successfully added cut record, enqueue preview job
	if _, _, errPreview := cutRecording.EnqueuePreviewsJob(); errPreview != nil {
		// Preview job enqueue failed - clean up the cut recording and file
		log.Errorf("Error enqueueing preview job for cut recording, cleaning up: %v", errPreview)

		if errDelete := cutRecording.DestroyRecording(); errDelete != nil {
			log.Errorf("Error cleaning up orphaned cut recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return errPreview
	}

	// The original file shall be deleted after the process if successful.
	if cutArgs.DeleteAfterCompletion {
		recording, err := database.FindRecordingByID(job.RecordingID)
		if err != nil {
			return err
		}
		return recording.DestroyRecording()
	}

	return nil
}

func DeleteJob(id uint) error {
	if err := database.DeleteJob(id); err != nil {
		return err
	}
	network.BroadCastClients(network.JobDeleteEvent, id)
	return nil
}

func StartJobProcessing() {
	ctxJobs, cancelJobs = context.WithCancel(context.Background())
	go processJobs(ctxJobs)
}

func StopJobProcessing() {
	cancelJobs()
}

func IsJobProcessing() bool {
	processingMutex.Lock()
	defer processingMutex.Unlock()
	return processing
}
