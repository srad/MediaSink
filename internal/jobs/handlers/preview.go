package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/srad/mediasink/config"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/util"
	"github.com/srad/mediasink/internal/ws"
)

// previewFramesHandler extracts preview frames from video recordings
type previewFramesHandler struct {
	deps *HandlerDependencies
}

var _ JobHandler = (*previewFramesHandler)(nil)

// NewPreviewFramesHandler creates a new preview frames handler
func NewPreviewFramesHandler(deps *HandlerDependencies) JobHandler {
	return &previewFramesHandler{
		deps: deps,
	}
}

// Name returns the handler name
func (h *previewFramesHandler) Name() string {
	return "preview_frames"
}

// Handle extracts preview frames from a video recording
func (h *previewFramesHandler) Handle(job *db.Job, threadCount int) error {
	// Delete existing preview entry and frames before regenerating
	_ = db.DeleteVideoPreviewByRecordingID(job.Recording.RecordingID)

	previewFramesPath := job.Recording.RecordingID.GetPreviewFramesPath(job.ChannelName)

	// Remove existing preview frames directory
	if err := os.RemoveAll(previewFramesPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing existing preview frames directory: %w", err)
	}

	// Create preview frames directory
	if err := os.MkdirAll(previewFramesPath, 0777); err != nil {
		return fmt.Errorf("error creating preview frames directory: %w", err)
	}

	// Calculate frame interval based on video duration (tiered approach)
	// Up to 6 hours: 1 frame/2 sec
	// 6 to 12 hours: 1 frame/3 sec
	// Above 12 hours: 1 frame/4 sec
	var frameInterval uint64

	switch {
	case job.Recording.Duration <= 6*3600: // Up to 6 hours
		frameInterval = 2
	case job.Recording.Duration <= 12*3600: // 6 to 12 hours
		frameInterval = 3
	default: // Above 12 hours
		frameInterval = 4
	}

	frameCount := uint64((job.Recording.Duration + float64(frameInterval) - 1) / float64(frameInterval))

	// Use select filter to extract frames at exact time intervals
	// This is more reliable than fps filter as it extracts at precise timestamps
	// Scale to max height with proportional width (-1 maintains aspect ratio)
	selectFilter := fmt.Sprintf("select='isnan(prev_selected_t)+gte(t-prev_selected_t\\,%d)',setpts=N/FRAME_RATE/TB,scale=-1:%d", frameInterval, config.FrameHeight)
	tempPattern := filepath.Join(previewFramesPath, "frame-%06d.jpg")

	// Execute FFmpeg to extract frames with progress tracking
	err := util.ExecSync(&util.ExecArgs{
		Command: "ffmpeg",
		CommandArgs: []string{
			"-i", job.Recording.AbsoluteChannelFilepath(),
			"-y",
			"-progress", "pipe:1",
			"-threads", fmt.Sprint(threadCount),
			"-an",
			"-vf", selectFilter,
			"-vsync", "vfr", // Variable frame rate - only output selected frames
			"-q:v", "2",
			"-hide_banner",
			"-loglevel", "error",
			tempPattern,
		},
		OnStart: func(info util.CommandInfo) {
			if err := job.UpdateInfo(info.Pid, info.Command); err != nil {
				h.deps.Logger.Errorf("[PreviewFrames] Error updating job info: %v", err)
			}
			ws.BroadCastClients(ws.JobStartEvent, JobMessage[util.TaskInfo]{
				Job: job,
				Data: util.TaskInfo{
					Pid:     info.Pid,
					Command: info.Command,
					Message: "Generating preview frames",
				},
			})
		},
		OnPipeOut: func(pm util.PipeMessage) {
			// Use the same parsing as enhance video job
			EmitProgressFromFrame(job, pm.Output, frameCount)

			kvs := util.ParseFFmpegKVs(pm.Output)
			if progress, ok := kvs["progress"]; ok {
				if progress == "end" {
					ws.BroadCastClients(ws.JobDoneEvent, JobMessage[util.TaskComplete]{
						Data: util.TaskComplete{Message: "Preview frames generated"},
						Job:  job,
					})
				}
			}
		},
		OnPipeErr: func(pm util.PipeMessage) {
			h.deps.Logger.Errorf("[PreviewFrames] Error: %s", pm.Output)
			ws.BroadCastClients(ws.JobErrorEvent, JobMessage[string]{Job: job, Data: pm.Output})
		},
	})

	if err != nil {
		return fmt.Errorf("error generating preview frames: %w", err)
	}

	// Rename frames using calculated timestamps
	// With select filter and regular intervals:
	// - frame-000001.jpg corresponds to timestamp 0
	// - frame-000002.jpg corresponds to timestamp frameInterval
	// - frame-000003.jpg corresponds to timestamp frameInterval * 2
	// etc.
	files, err := os.ReadDir(previewFramesPath)
	if err != nil {
		return fmt.Errorf("error reading preview frames directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".jpg") {
			// Extract frame number from filename (e.g., "frame-000001.jpg" -> 1)
			var frameNum int64
			_, err := fmt.Sscanf(file.Name(), "frame-%06d.jpg", &frameNum)
			if err != nil {
				// Skip files that don't match pattern
				continue
			}

			// Calculate timestamp: frame N (1-indexed) corresponds to timestamp (N-1) * frameInterval
			// Frame 1 → 0 seconds, Frame 2 → frameInterval seconds, Frame 3 → frameInterval*2, etc.
			timestamp := uint64(frameNum-1) * frameInterval
			newName := fmt.Sprintf("%d.jpg", timestamp)
			oldPath := filepath.Join(previewFramesPath, file.Name())
			newPath := filepath.Join(previewFramesPath, newName)

			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("error renaming preview frame: %w", err)
			}
		}
	}

	// Store preview metadata in video_previews table
	relativePreviewPath := job.Recording.RecordingID.GetRelativePreviewFramesPath(job.ChannelName)
	videoPreview := &db.VideoPreview{
		RecordingID:   job.Recording.RecordingID,
		FrameCount:    frameCount,
		FrameInterval: frameInterval,
		PreviewPath:   relativePreviewPath,
	}

	// Create new preview record
	if err := videoPreview.CreateVideoPreview(); err != nil {
		return err
	}

	// Check if the job was canceled externally before enqueuing analysis job
	isCanceled, err := db.JobHasStatus(job.JobID, db.StatusJobCanceled)
	if err != nil {
		h.deps.Logger.Warnf("[PreviewFrames] Failed to check job cancellation status: %v", err)
		// Continue anyway - this is a warning level issue
	} else if isCanceled {
		// Job was canceled externally - don't enqueue analysis
		h.deps.Logger.Infof("[PreviewFrames] Preview job was canceled externally, skipping analysis job enqueue")
		return nil
	}

	// Automatically enqueue analysis job after preview frames are generated
	if _, errAnalysis := job.Recording.EnqueueAnalysisJob(); errAnalysis != nil {
		h.deps.Logger.Warnf("[PreviewFrames] Failed to enqueue analysis job: %v (preview frames were generated successfully)", errAnalysis)
		// Don't fail the preview job if analysis enqueue fails - previews are still useful without analysis
		// Just log the warning and continue
	}

	return nil
}
