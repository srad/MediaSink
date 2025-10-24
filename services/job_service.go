package services

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/srad/mediasink/conf"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"
	"github.com/srad/mediasink/models/responses"
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
					Job:  job,
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
	case database.TaskMerge:
		return handleJob(job, processMerge(job))
	case database.TaskEnhanceVideo:
		return handleJob(job, processEnhanceVideo(job))
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
			log.Errorf("error deleting file %s: %v", result.Filepath, errDelete)
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

func processMerge(job *database.Job) error {
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

	log.Infof("[MergeJob] Starting merge of %d recordings", len(recordingIDs))

	var errMerge error
	if reEncode {
		// Re-encoded merge
		errMerge = helpers.MergeVideosReEncoded(&helpers.MergeReEncodeArgs{
			OnStart: func(info helpers.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					log.Errorf("[MergeJob] Error updating job info: %v", err)
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
			log.Errorf("[MergeJob] Error writing concat file: %v", errWrite)
			return fmt.Errorf("error writing concat file: %w", errWrite)
		}

		errMerge = helpers.MergeVideos(&helpers.MergeArgs{
			OnStart: func(info helpers.CommandInfo) {
				if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
					log.Errorf("[MergeJob] Error updating job info: %v", err)
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
			log.Warnf("[MergeJob] Error deleting concat file: %v", errCleanup)
		}
	}

	if errMerge != nil {
		log.Errorf("[MergeJob] Error merging recordings: %v", errMerge)
		return errMerge
	}

	// Verify merged output
	outputVideo := &helpers.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("[MergeJob] Error deleting unreadable merged file: %v", errCleanup)
		}
		return fmt.Errorf("error validating merged video: %w", err)
	}

	// Create recording entry in database
	mergedRecording, errCreate := database.CreateRecording(job.ChannelID, filename, "merged")
	if errCreate != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("[MergeJob] Error deleting merged file after DB save failure: %v", errCleanup)
		}
		return fmt.Errorf("error creating merged recording in database: %w", errCreate)
	}

	// Enqueue preview jobs for merged recording
	if _, _, errPreview := mergedRecording.EnqueuePreviewsJob(); errPreview != nil {
		log.Errorf("[MergeJob] Error enqueueing preview jobs for merged recording: %v", errPreview)

		if errDelete := mergedRecording.DestroyRecording(); errDelete != nil {
			log.Errorf("[MergeJob] Error cleaning up orphaned merged recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return fmt.Errorf("error enqueueing preview job: %w", errPreview)
	}

	log.Infof("[MergeJob] Successfully merged %d recordings into %s", len(recordingIDs), filename)
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

func emitProgressFromFrame(job *database.Job, s string, totalCount uint64) {
	if strings.Contains(s, "frame=") {
		s := strings.Split(s, "=")
		if len(s) == 2 {
			if p, err := strconv.ParseUint(s[1], 10, 64); err == nil {
				network.BroadCastClients(network.JobProgressEvent, JobMessage[helpers.TaskProgress]{Job: job, Data: helpers.TaskProgress{
					Step:    1,
					Steps:   1,
					Total:   totalCount,
					Current: p,
				}})
			}
		}
	}
}

func processEnhanceVideo(job *database.Job) error {
	enhanceArgs, err := database.UnmarshalJobArg[helpers.EnhanceArgs](job)
	if err != nil {
		return err
	}

	// Validate encoding preset
	if !enhanceArgs.EncodingPreset.Validate() {
		return fmt.Errorf("invalid encoding preset: %s", enhanceArgs.EncodingPreset)
	}

	inputFile := job.Recording.AbsoluteChannelFilepath()
	log.Infof("[EnhanceVideoJob] Starting enhancement of '%s' to %s with preset %s", job.Filename, enhanceArgs.TargetResolution, enhanceArgs.EncodingPreset)

	// Get input video info to check current resolution
	inputVideo := &helpers.Video{FilePath: inputFile}
	inputInfo, err := inputVideo.GetVideoInfo()
	if err != nil {
		return fmt.Errorf("error reading input video info: %w", err)
	}

	// Generate output filename
	now := time.Now()
	stamp := now.Format("2006_01_02_15_04_05")
	filename := database.RecordingFileName(fmt.Sprintf("%s_enhanced_%s_%s.mp4", job.ChannelName, enhanceArgs.TargetResolution, stamp))
	outputFile := job.ChannelName.AbsoluteChannelFilePath(filename)

	// Get target dimensions
	targetWidth, targetHeight := enhanceArgs.TargetResolution.GetDimensions()

	// Build FFmpeg filter chain
	filterChain := fmt.Sprintf("hqdn3d=luma_spatial=%f:chroma_spatial=%f", enhanceArgs.DenoiseStrength, enhanceArgs.DenoiseStrength/2)

	// Only scale if input resolution differs from target
	if inputInfo.Width != targetWidth || inputInfo.Height != targetHeight {
		filterChain += fmt.Sprintf(",scale=%d:%d:flags=lanczos", targetWidth, targetHeight)
	}

	// Always apply sharpening
	filterChain += fmt.Sprintf(",unsharp=lx=5:ly=5:la=%f:cx=5:cy=5:ca=%f", enhanceArgs.SharpenStrength, enhanceArgs.SharpenStrength)

	// Add normalization if requested
	if enhanceArgs.ApplyNormalize {
		filterChain += ",normalize=independence=0"
	}

	cmdArgs := []string{
		"-y",
		"-progress", "pipe:1",
		"-i", inputFile,
		"-hide_banner",
		"-threads", fmt.Sprint(conf.ThreadCount),
		"-loglevel", "error",
		"-stats_period", "2.0",
		"-vf", filterChain + ",format=yuv420p",
		"-c:v", "libx265",
		"-crf", fmt.Sprintf("%d", enhanceArgs.CRF),
		"-preset", string(enhanceArgs.EncodingPreset),
		"-pix_fmt", "yuv420p",
		"-c:a", "copy",
		"-movflags", "faststart",
		outputFile,
	}

	err = helpers.ExecSync(&helpers.ExecArgs{
		Command:     "ffmpeg",
		CommandArgs: cmdArgs,
		OnStart: func(info helpers.CommandInfo) {
			if errUpdate := job.UpdateInfo(info.Pid, info.Command); errUpdate != nil {
				log.Errorf("[EnhanceVideoJob] Error updating job info: %v", errUpdate)
			}
			network.BroadCastClients(network.JobStartEvent, JobMessage[helpers.TaskInfo]{
				Job: job,
				Data: helpers.TaskInfo{
					Pid:     info.Pid,
					Command: info.Command,
					Message: fmt.Sprintf("Enhancing to %s with preset %s", enhanceArgs.TargetResolution, enhanceArgs.EncodingPreset),
				},
			})
		},
		OnPipeOut: func(pm helpers.PipeMessage) {
			emitProgressFromFrame(job, pm.Output, inputInfo.PacketCount)
		},
		OnPipeErr: func(info helpers.PipeMessage) {
			// Only log actual errors, not x265 info/warnings or benign permission issues
			if !strings.Contains(info.Output, "x265 [info]") &&
				!strings.Contains(info.Output, "x265 [warning]") &&
				!strings.Contains(info.Output, "set_mempolicy") {
				log.Errorf("[EnhanceVideoJob] Error: %s", info.Output)
				network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: info.Output})
			}
		},
	})

	if err != nil {
		log.Errorf("[EnhanceVideoJob] Error enhancing video: %v", err)
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("[EnhanceVideoJob] Error deleting failed output file: %v", errCleanup)
		}
		return fmt.Errorf("error enhancing video: %w", err)
	}

	// Verify enhanced output
	outputVideo := &helpers.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("[EnhanceVideoJob] Error deleting unreadable enhanced file: %v", errCleanup)
		}
		return fmt.Errorf("error validating enhanced video: %w", err)
	}

	// Create recording entry in database
	enhancedRecording, errCreate := database.CreateRecording(job.ChannelID, filename, "enhanced")
	if errCreate != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			log.Warnf("[EnhanceVideoJob] Error deleting enhanced file after DB save failure: %v", errCleanup)
		}
		return fmt.Errorf("error creating enhanced recording in database: %w", errCreate)
	}

	// Enqueue preview jobs for enhanced recording
	if _, _, errPreview := enhancedRecording.EnqueuePreviewsJob(); errPreview != nil {
		log.Errorf("[EnhanceVideoJob] Error enqueueing preview jobs: %v", errPreview)

		if errDelete := enhancedRecording.DestroyRecording(); errDelete != nil {
			log.Errorf("[EnhanceVideoJob] Error cleaning up orphaned enhanced recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return fmt.Errorf("error enqueueing preview job: %w", errPreview)
	}

	log.Infof("[EnhanceVideoJob] Successfully enhanced '%s' to %s", job.Filename, filename)
	return nil
}

// GetEnhancementDescriptions returns descriptions for all enhancement parameters
func GetEnhancementDescriptions() *responses.EnhancementDescriptions {
	return &responses.EnhancementDescriptions{
		Presets: [7]responses.PresetDescription{
			{
				Preset:      "veryfast",
				Label:       "Very Fast",
				Description: "Encodes very quickly, larger file size, minimal optimization",
				EncodeSpeed: "~30-50 min per hour of video",
			},
			{
				Preset:      "faster",
				Label:       "Faster",
				Description: "Fast encoding with good compression balance",
				EncodeSpeed: "~20-30 min per hour of video",
			},
			{
				Preset:      "fast",
				Label:       "Fast",
				Description: "Balanced speed and compression",
				EncodeSpeed: "~15-20 min per hour of video",
			},
			{
				Preset:      "medium",
				Label:       "Medium",
				Description: "Default preset, very good compression efficiency",
				EncodeSpeed: "~8-12 min per hour of video",
			},
			{
				Preset:      "slow",
				Label:       "Slow",
				Description: "Slower encoding, excellent compression",
				EncodeSpeed: "~4-6 min per hour of video",
			},
			{
				Preset:      "slower",
				Label:       "Slower",
				Description: "Very slow encoding, best compression efficiency",
				EncodeSpeed: "~2-3 min per hour of video",
			},
			{
				Preset:      "veryslow",
				Label:       "Very Slow",
				Description: "Extremely slow, maximum compression, best quality/size ratio",
				EncodeSpeed: "~1-2 min per hour of video",
			},
		},
		CRFValues: [5]responses.CRFDescription{
			{
				Value:       15,
				Label:       "CRF 15 - Highest Quality",
				Description: "Near-lossless quality, largest file size, excellent for archival and professional use",
				Quality:     "Visually lossless",
				ApproxRatio: 0.38,
			},
			{
				Value:       18,
				Label:       "CRF 18 - High Quality (Recommended)",
				Description: "High quality with good compression, ~42% of original file size, recommended default",
				Quality:     "Very high quality",
				ApproxRatio: 0.42,
			},
			{
				Value:       22,
				Label:       "CRF 22 - Balanced",
				Description: "Good balance between quality and file size, ~55% of original, suitable for most uses",
				Quality:     "Good quality",
				ApproxRatio: 0.55,
			},
			{
				Value:       25,
				Label:       "CRF 25 - Smaller Files",
				Description: "Noticeable quality reduction, ~68% of original file size, for storage-constrained scenarios",
				Quality:     "Acceptable quality",
				ApproxRatio: 0.68,
			},
			{
				Value:       28,
				Label:       "CRF 28 - Lowest Quality",
				Description: "Significant quality loss, smallest file size (~80%), only for previews or when space is critical",
				Quality:     "Low quality",
				ApproxRatio: 0.80,
			},
		},
		Resolutions: [4]responses.ResolutionDescription{
			{
				Resolution:  "720p",
				Dimensions:  "1280x720",
				Description: "HD quality, suitable for small screens and streaming",
				UseCase:     "Mobile devices, tablets, web streaming",
			},
			{
				Resolution:  "1080p",
				Dimensions:  "1920x1080",
				Description: "Full HD quality, standard for most modern displays",
				UseCase:     "Desktop monitors, laptops, streaming (Recommended)",
			},
			{
				Resolution:  "1440p",
				Dimensions:  "2560x1440",
				Description: "QHD quality, sharper than 1080p, good for high-end displays",
				UseCase:     "High-resolution monitors, premium viewing",
			},
			{
				Resolution:  "4k",
				Dimensions:  "3840x2160",
				Description: "Ultra HD quality, 4 times the pixels of 1080p, largest file size",
				UseCase:     "4K monitors, professional use, archival",
			},
		},
		Filters: responses.FilterDescriptions{
			DenoiseStrength: responses.FilterDescription[float64]{
				Name:        "Denoise Strength",
				Description: "Reduces video noise/grain. Higher values remove more noise but may blur fine details",
				Recommended: 4.0,
				Range:       "1.0 - 10.0",
				MinValue:    1.0,
				MaxValue:    10.0,
			},
			SharpenStrength: responses.FilterDescription[float64]{
				Name:        "Sharpen Strength",
				Description: "Enhances edges and details. Higher values create more defined edges but may introduce artifacts",
				Recommended: 1.25,
				Range:       "0.0 - 2.0",
				MinValue:    0.0,
				MaxValue:    2.0,
			},
			ApplyNormalize: responses.FilterDescription[bool]{
				Name:        "Auto Color/Brightness Correction",
				Description: "Automatically adjusts brightness and color levels to improve overall appearance",
				Recommended: true,
				Range:       "true/false",
				MinValue:    false,
				MaxValue:    true,
			},
		},
	}
}

// EstimateEnhancementFileSize estimates the output file size for video enhancement
func EstimateEnhancementFileSize(recording *database.Recording, targetRes helpers.ResolutionType, crf uint) (int64, error) {
	if recording == nil {
		return 0, fmt.Errorf("recording is nil")
	}

	// Get input file size
	inputFileSize := int64(recording.Size)
	if inputFileSize == 0 {
		return 0, fmt.Errorf("cannot estimate: input file size is 0")
	}

	// Get target dimensions
	targetWidth, targetHeight := targetRes.GetDimensions()

	// Calculate resolution scaling factor
	// If upscaling (resolution increases), file size increases
	// If downscaling (resolution decreases), file size decreases
	currentPixels := uint64(recording.Width) * uint64(recording.Height)
	targetPixels := uint64(targetWidth) * uint64(targetHeight)

	var resolutionFactor float64 = 1.0
	if currentPixels > 0 {
		resolutionFactor = float64(targetPixels) / float64(currentPixels)
	}

	// CRF compression ratios (x265 encoding efficiency)
	// Based on empirical data for typical video content
	var crfFactor float64
	switch {
	case crf >= 15 && crf <= 17:
		crfFactor = 0.38 // ~38% of original (high quality)
	case crf >= 18 && crf <= 19:
		crfFactor = 0.42 // ~42% of original (standard quality)
	case crf >= 20 && crf <= 22:
		crfFactor = 0.55 // ~55% of original (balanced)
	case crf >= 23 && crf <= 25:
		crfFactor = 0.68 // ~68% of original (smaller files)
	case crf >= 26:
		crfFactor = 0.80 // ~80% of original (low quality)
	default:
		crfFactor = 0.42
	}

	// Calculate estimated output file size
	estimatedSize := int64(float64(inputFileSize) * resolutionFactor * crfFactor)

	return estimatedSize, nil
}

func IsJobProcessing() bool {
	processingMutex.Lock()
	defer processingMutex.Unlock()
	return processing
}
