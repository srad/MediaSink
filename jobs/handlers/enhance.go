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

// enhanceHandler enhances videos with filters, scaling, and encoding optimizations
type enhanceHandler struct {
	deps *HandlerDependencies
}

var _ JobHandler = (*enhanceHandler)(nil)

// NewEnhanceHandler creates a new enhance handler
func NewEnhanceHandler(deps *HandlerDependencies) JobHandler {
	return &enhanceHandler{
		deps: deps,
	}
}

// Name returns the handler name
func (h *enhanceHandler) Name() string {
	return "enhance"
}

// Handle enhances a video with filters, scaling, and encoding optimizations
func (h *enhanceHandler) Handle(job *database.Job, threadCount int) error {
	enhanceArgs, err := database.UnmarshalJobArg[helpers.EnhanceArgs](job)
	if err != nil {
		return err
	}

	// Validate encoding preset
	if !enhanceArgs.EncodingPreset.Validate() {
		return fmt.Errorf("invalid encoding preset: %s", enhanceArgs.EncodingPreset)
	}

	inputFile := job.Recording.AbsoluteChannelFilepath()
	h.deps.Logger.Infof("[EnhanceVideoJob] Starting enhancement of '%s' to %s with preset %s", job.Filename, enhanceArgs.TargetResolution, enhanceArgs.EncodingPreset)

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
		"-threads", fmt.Sprint(threadCount),
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
				h.deps.Logger.Errorf("[EnhanceVideoJob] Error updating job info: %v", errUpdate)
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
			EmitProgressFromFrame(job, pm.Output, inputInfo.PacketCount)
		},
		OnPipeErr: func(info helpers.PipeMessage) {
			// Only log actual errors, not x265 info/warnings or benign permission issues
			if !strings.Contains(info.Output, "x265 [info]") &&
				!strings.Contains(info.Output, "x265 [warning]") &&
				!strings.Contains(info.Output, "set_mempolicy") {
				h.deps.Logger.Errorf("[EnhanceVideoJob] Error: %s", info.Output)
				network.BroadCastClients(network.JobErrorEvent, JobMessage[string]{Job: job, Data: info.Output})
			}
		},
	})

	if err != nil {
		h.deps.Logger.Errorf("[EnhanceVideoJob] Error enhancing video: %v", err)
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("[EnhanceVideoJob] Error deleting failed output file: %v", errCleanup)
		}
		return fmt.Errorf("error enhancing video: %w", err)
	}

	// Verify enhanced output
	outputVideo := &helpers.Video{FilePath: outputFile}
	if _, err := outputVideo.GetVideoInfo(); err != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("[EnhanceVideoJob] Error deleting unreadable enhanced file: %v", errCleanup)
		}
		return fmt.Errorf("error validating enhanced video: %w", err)
	}

	// Create recording entry in database
	enhancedRecording, errCreate := database.CreateRecording(job.ChannelID, filename, "enhanced")
	if errCreate != nil {
		if errCleanup := os.Remove(outputFile); errCleanup != nil {
			h.deps.Logger.Warnf("[EnhanceVideoJob] Error deleting enhanced file after DB save failure: %v", errCleanup)
		}
		return fmt.Errorf("error creating enhanced recording in database: %w", errCreate)
	}

	// Enqueue preview jobs for enhanced recording
	if _, errPreview := enhancedRecording.EnqueuePreviewFramesJob(); errPreview != nil {
		h.deps.Logger.Errorf("[EnhanceVideoJob] Error enqueueing preview jobs: %v", errPreview)

		if errDelete := enhancedRecording.DestroyRecording(); errDelete != nil {
			h.deps.Logger.Errorf("[EnhanceVideoJob] Error cleaning up orphaned enhanced recording: %v", errDelete)
			return fmt.Errorf("error enqueueing preview job: %w (also failed to cleanup: %w)", errPreview, errDelete)
		}
		return fmt.Errorf("error enqueueing preview job: %w", errPreview)
	}

	h.deps.Logger.Infof("[EnhanceVideoJob] Successfully enhanced '%s' to %s", job.Filename, filename)
	return nil
}
