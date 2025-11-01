package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego/utils"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/conf"
)

var (
	VideosFolder        = "videos"
	PreviewFramesFolder = "frames"
)

// Video Represent a video to which operations can be applied.
type Video struct {
	FilePath string `validate:"required,filepath"`
}

type CuttingJob struct {
	OnStart    func(*CommandInfo)
	OnProgress func(string)
}

type CutArgs struct {
	Starts                []string `json:"starts"`
	Ends                  []string `json:"ends"`
	DeleteAfterCompletion bool     `json:"deleteAfterCut"`
}

type MergeJobArgs struct {
	RecordingIDs []uint `json:"recordingIds"`
	ReEncode     bool   `json:"reEncode"`
}

type ResolutionType string

const (
	Resolution720p  ResolutionType = "720p"
	Resolution1080p ResolutionType = "1080p"
	Resolution1440p ResolutionType = "1440p"
	Resolution4K    ResolutionType = "4k"
)

// GetResolutionDimensions returns width and height for a resolution type
func (r ResolutionType) GetDimensions() (width uint, height uint) {
	switch r {
	case Resolution720p:
		return 1280, 720
	case Resolution1080p:
		return 1920, 1080
	case Resolution1440p:
		return 2560, 1440
	case Resolution4K:
		return 3840, 2160
	default:
		return 1920, 1080
	}
}

type EncodingPreset string

const (
	PresetVeryFast  EncodingPreset = "veryfast"
	PresetFaster    EncodingPreset = "faster"
	PresetFast      EncodingPreset = "fast"
	PresetMedium    EncodingPreset = "medium"
	PresetSlow      EncodingPreset = "slow"
	PresetSlower    EncodingPreset = "slower"
	PresetVerySlow  EncodingPreset = "veryslow"
)

// Validate checks if the preset is valid
func (p EncodingPreset) Validate() bool {
	switch p {
	case PresetVeryFast, PresetFaster, PresetFast, PresetMedium, PresetSlow, PresetSlower, PresetVerySlow:
		return true
	default:
		return false
	}
}

type EnhanceArgs struct {
	TargetResolution ResolutionType `json:"targetResolution"`
	DenoiseStrength  float64        `json:"denoiseStrength"`
	SharpenStrength  float64        `json:"sharpenStrength"`
	ApplyNormalize   bool           `json:"applyNormalize"`
	EncodingPreset   EncodingPreset `json:"encodingPreset"`
	CRF              uint           `json:"crf"`
}

type TaskProgress struct {
	Current uint64 `json:"current"`
	Total   uint64 `json:"total"`
	Steps   uint   `json:"steps"`
	Step    uint   `json:"step"`
	Message string `json:"message"`
}

type TaskComplete struct {
	Steps   uint   `json:"steps"`
	Step    uint   `json:"step"`
	Message string `json:"message"`
}

type TaskInfo struct {
	Steps   uint   `json:"steps"`
	Step    uint   `json:"step"`
	Pid     int    `json:"pid"`
	Command string `json:"command"`
	Message string `json:"message"`
}

type VideoConversionArgs struct {
	OnStart     func(info TaskInfo)
	OnProgress  func(info TaskProgress)
	OnEnd       func(task TaskComplete)
	OnError     func(error)
	InputPath   string
	OutputPath  string
	Filename    string
	ThreadCount int
}

type ProcessInfo struct {
	JobType string
	Frame   uint64
	Total   int
	Raw     string
}

type JSONFFProbeInfo struct {
	Streams []struct {
		Width       uint   `json:"width"`
		Height      uint   `json:"height"`
		RFrameRate  string `json:"r_frame_rate"`
		PacketCount string `json:"nb_read_packets"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
}

type FFProbeInfo struct {
	Fps         float64
	Duration    float64
	Size        uint64
	BitRate     uint64
	Width       uint
	Height      uint
	PacketCount uint64
}

type ConversionResult struct {
	ChannelName string
	Filename    string
	Filepath    string
	CreatedAt   time.Time
}

type PreviewResult struct {
	FilePath string
	Filename string
}

type MergeArgs struct {
	OnStart                func(info CommandInfo)
	OnProgress             func(info PipeMessage)
	OnErr                  func(error)
	MergeFileAbsolutePath  string
	AbsoluteOutputFilepath string
}

type MergeReEncodeArgs struct {
	OnStart                func(info CommandInfo)
	OnProgress             func(info PipeMessage)
	OnErr                  func(error)
	InputFiles             []string
	AbsoluteOutputFilepath string
}

func ExtractFirstFrame(input, outputPathPoster string) error {
	err := ExecSync(&ExecArgs{
		Command:     "ffmpeg",
		CommandArgs: []string{"-y", "-hide_banner", "-loglevel", "error", "-i", input, "-r", "1", "-vf", fmt.Sprintf("scale=-1:%d", conf.FrameHeight), "-q:v", "2", "-frames:v", "1", outputPathPoster},
	})

	if err != nil {
		return fmt.Errorf("error extracting frame '%s'", err)
	}

	return nil
}

func calcFps(output string) (float64, error) {
	numbers := strings.Split(output, "/")

	if len(numbers) != 2 {
		return 0, errors.New("ffprobe output is not as expected a divison: a/b")
	}

	a, err := strconv.ParseFloat(numbers[0], 32)
	if err != nil {
		return 0, err
	}

	b, err := strconv.ParseFloat(numbers[1], 32)
	if err != nil {
		return 0, err
	}

	fps := a / b

	return fps, nil
}

func ConvertVideo(args *VideoConversionArgs, mediaType string) (*ConversionResult, error) {
	input := filepath.Join(args.OutputPath, args.Filename)
	if !utils.FileExists(input) {
		return nil, fmt.Errorf("file '%s' does not exit", input)
	}

	// Might seem redundant, but since we have no dependent types...
	if mediaType == "mp3" {
		mp3Filename := fmt.Sprintf("%s.mp3", FileNameWithoutExtension(args.Filename))
		outputAbsoluteMp3 := filepath.Join(args.OutputPath, mp3Filename)

		result := &ConversionResult{
			Filename:  mp3Filename,
			CreatedAt: time.Now(),
			Filepath:  outputAbsoluteMp3,
		}

		err := ExecSync(&ExecArgs{
			OnPipeErr: func(info PipeMessage) {
				if args.OnError != nil {
					args.OnError(errors.New(info.Output))
				}
			},
			OnStart: func(info CommandInfo) {
				args.OnStart(TaskInfo{
					Steps:   3,
					Pid:     info.Pid,
					Command: info.Command,
				})
			},
			OnPipeOut: func(message PipeMessage) {
				kvs := ParseFFmpegKVs(message.Output)

				if frame, ok := kvs["frame"]; ok {
					if value, err := strconv.ParseUint(frame, 10, 64); err == nil {
						args.OnProgress(TaskProgress{Current: value})
					}
				}
				if progress, ok := kvs["progress"]; ok {
					if progress == "end" && args.OnEnd != nil {
						args.OnEnd(TaskComplete{
							Steps: 1,
							Step:  1,
						})
					} else {
						fmt.Println(progress)
					}
				}
			},
			Command:     "ffmpeg",
			CommandArgs: []string{"-i", input, "-y", "-threads", fmt.Sprint(args.ThreadCount), "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-q:a", "0", "-map", "a", outputAbsoluteMp3},
		})

		return result, err
	}

	// Create new filename
	name := fmt.Sprintf("%s_%s.mp4", FileNameWithoutExtension(args.Filename), mediaType)
	output := filepath.Join(args.OutputPath, name)

	result := &ConversionResult{
		Filename:  name,
		CreatedAt: time.Now(),
		Filepath:  output,
	}

	err := ExecSync(&ExecArgs{
		OnPipeErr: func(info PipeMessage) {
			log.Error(info.Output)
		},
		OnStart: func(info CommandInfo) {
			args.OnStart(TaskInfo{
				Steps:   1,
				Pid:     info.Pid,
				Command: info.Command,
			})
		},
		Command: "ffmpeg",
		// Preset values: https://trac.ffmpeg.org/wiki/Encode/H.264
		// ultrafast
		// superfast
		// veryfast
		// faster
		// fast
		// medium – default preset
		// slow
		// slower
		// veryslow
		CommandArgs: []string{"-i", input, "-y", "-threads", fmt.Sprint(args.ThreadCount), "-an", "-vf", fmt.Sprintf("scale=-1:%s", mediaType), "-hide_banner", "-loglevel", "error", "-progress", "pipe:1", "-movflags", "faststart", "-c:v", "libx264", "-crf", "18", "-preset", "medium", "-c:a", "copy", output},
	})

	return result, err
}

// ExecPreviewFrames generates individual preview frames for a video
// Frames are extracted at regular intervals and named with the timestamp in seconds
// previewFramesPath should be the absolute path from RecordingID.GetPreviewFramesPath()
func (video *Video) ExecPreviewFrames(args *VideoConversionArgs, videoDuration float64, previewFramesPath string) (*PreviewResult, error) {
	basename := filepath.Base(video.FilePath)
	filename := FileNameWithoutExtension(basename)

	// Calculate frame interval to ensure whole-number intervals
	// Start with 2-second intervals and increase if frameCount exceeds 800
	frameInterval := uint64(2)
	frameCount := uint64((videoDuration + float64(frameInterval) - 1) / float64(frameInterval)) // ceil division

	for frameCount > 800 {
		frameInterval++
		frameCount = uint64((videoDuration + float64(frameInterval) - 1) / float64(frameInterval))
	}

	// Ensure minimum of 30 frames
	if frameCount < 30 {
		frameInterval = uint64((videoDuration + 29) / 30) // ceil division
		frameCount = 30
	}

	// Remove existing preview frames directory to ensure clean regeneration
	previewDir := previewFramesPath
	if err := os.RemoveAll(previewDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error removing existing preview frames directory: %w", err)
	}

	// Create preview frames directory
	if err := os.MkdirAll(previewDir, 0777); err != nil {
		return nil, fmt.Errorf("error creating preview frames directory: %w", err)
	}

	// Calculate fps based on frame interval
	fps := 1.0 / float64(frameInterval)

	// Use a counter pattern for temp files
	tempPattern := filepath.Join(previewDir, "frame-%06d.jpg")

	// Track frame timestamps from FFmpeg progress output
	var framesExtracted uint64
	var lastOutTimeMs int64
	var lastFrameNum int64
	frameTimestamps := make(map[int64]int64) // frameNum -> timestamp in seconds

	err := ExecSync(&ExecArgs{
		OnStart: func(info CommandInfo) {
			if args.OnStart != nil {
				args.OnStart(TaskInfo{
					Pid:     info.Pid,
					Command: info.Command,
					Message: "Generating preview frames",
				})
			}
		},
		OnPipeOut: func(out PipeMessage) {
			kvs := ParseFFmpegKVs(out.Output)

			// Track the actual timestamp from FFmpeg
			if outTimeMs, ok := kvs["out_time_ms"]; ok {
				if value, err := strconv.ParseInt(outTimeMs, 10, 64); err == nil && value > 0 {
					lastOutTimeMs = value
				}
			}

			// When a new frame is extracted, record its timestamp
			if frame, ok := kvs["frame"]; ok {
				if value, err := strconv.ParseInt(frame, 10, 64); err == nil && value > 0 {
					if value > lastFrameNum {
						lastFrameNum = value
						// Store the timestamp in seconds (convert from milliseconds)
						frameTimestamps[value] = lastOutTimeMs / 1000
						framesExtracted++
						args.OnProgress(TaskProgress{
							Current: framesExtracted,
							Total:   frameCount,
							Message: "Generating preview frames",
						})
					}
				}
			}

			if progress, ok := kvs["progress"]; ok {
				if progress == "end" && args.OnEnd != nil {
					args.OnEnd(TaskComplete{
						Message: "Preview frames generated",
					})
				}
			}
		},
		OnPipeErr: func(pipe PipeMessage) {
			if args.OnError != nil {
				args.OnError(fmt.Errorf("error generating frames: %s", pipe.Output))
			}
		},
		Command: "ffmpeg",
		CommandArgs: []string{
			"-i", video.FilePath,
			"-y",
			"-progress", "pipe:1",
			"-threads", fmt.Sprint(conf.ThreadCount),
			"-an",
			"-vf", fmt.Sprintf("fps=%.4f", fps),
			"-q:v", "2",
			"-hide_banner",
			"-loglevel", "error",
			tempPattern,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("error generating preview frames for '%s': %w", video.FilePath, err)
	}

	// Rename frames using the actual timestamps from FFmpeg
	files, err := os.ReadDir(previewDir)
	if err != nil {
		return nil, fmt.Errorf("error reading preview frames directory: %w", err)
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

			// Get the actual timestamp for this frame from our tracking map
			if timestamp, ok := frameTimestamps[frameNum]; ok {
				newName := fmt.Sprintf("%d.jpg", timestamp)
				oldPath := filepath.Join(previewDir, file.Name())
				newPath := filepath.Join(previewDir, newName)

				if err := os.Rename(oldPath, newPath); err != nil {
					return nil, fmt.Errorf("error renaming preview frame: %w", err)
				}
			}
		}
	}

	return &PreviewResult{FilePath: previewDir, Filename: filename}, nil
}

// CreatePreviewShots Create a separate preview image file, at every frame distance.
//func (video *Video) CreatePreviewShots(errListener func(s string), outputDir string, filename string, frameDistance uint, frameHeight uint, fps float64) (string, error) {
//	dirPreview := filepath.Join(outputDir, conf.ScreensFolder, filename)
//	if err := os.MkdirAll(dirPreview, 0777); err != nil {
//		return dirPreview, err
//	}
//
//	outFile := fmt.Sprintf("%s_%%010d.jpg", filename)
//
//	return dirPreview, ExecSync(&ExecArgs{
//		OnPipeErr: func(info PipeMessage) {
//			errListener(info.Output)
//		},
//		Command:     "ffmpeg",
//		CommandArgs: []string{"-i", video.AbsoluteChannelFilepath, "-y", "-progress", "pipe:1", "-q:v", "0", "-threads", fmt.Sprint(conf.ThreadCount), "-an", "-vf", fmt.Sprintf("select=not(mod(n\\,%d)),scale=-2:%d", frameDistance, frameHeight), "-hide_banner", "-loglevel", "error", "-stats", "-fps_mode", "vfr", filepath.Join(dirPreview, outFile)},
//	})
//}

// GetFrameCount This requires an entire video passthrough
//func (video *Video) GetFrameCount() (uint64, error) {
//	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=nb_read_packets", "-of", "csv=p=0", "-select_streams", "v:0", "-count_packets", video.FilePath)
//	stdout, err := cmd.CombinedOutput()
//	output := strings.TrimSpace(string(stdout))
//
//	if err != nil {
//		return 0, fmt.Errorf("error getting frame count for '%s': %s", video.FilePath, stdout)
//	}
//
//	fps, err := strconv.ParseUint(output, 10, 64)
//	if err != nil {
//		return 0, nil
//	}
//
//	return fps, nil
//}

// GetVideoInfo Generate file information via ffprobe in JSON and parses it from stout.
func (video *Video) GetVideoInfo() (*FFProbeInfo, error) {
	cmd := exec.Command("ffprobe", "-i", video.FilePath, "-show_entries", "format=bit_rate,size,duration", "-show_entries", "stream=r_frame_rate,width,height,nb_read_packets", "-v", "error", "-select_streams", "v:0", "-count_packets", "-of", "default=noprint_wrappers=1", "-print_format", "json")
	stdout, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(stdout))

	if err != nil {
		return nil, fmt.Errorf("error ffprobe: %s: %s", err, output)
	}

	parsed := &JSONFFProbeInfo{}
	err = json.Unmarshal([]byte(output), &parsed)
	if err != nil {
		return nil, err
	}

	info := &FFProbeInfo{
		BitRate:     0,
		Size:        0,
		Height:      0,
		Width:       0,
		Duration:    0,
		Fps:         0,
		PacketCount: 0,
	}

	duration, err := strconv.ParseFloat(parsed.Format.Duration, 64)
	if err != nil {
		return info, err
	}
	info.Duration = duration

	bitrate, err := strconv.ParseUint(parsed.Format.BitRate, 10, 64)
	if err != nil {
		return info, err
	}
	info.BitRate = bitrate

	size, err := strconv.ParseUint(parsed.Format.Size, 10, 64)
	if err != nil {
		return info, err
	}
	info.Size = size

	fps, err := calcFps(parsed.Streams[0].RFrameRate)
	if err != nil {
		return info, err
	}
	info.Fps = fps

	packets, err := strconv.ParseUint(parsed.Streams[0].PacketCount, 10, 64)
	if err != nil {
		return info, err
	}
	info.PacketCount = packets

	info.Width = parsed.Streams[0].Width
	info.Height = parsed.Streams[0].Height

	return info, nil
}

func MergeVideos(args *MergeArgs) error {
	log.Infoln("---------------------------------------------- Merge Job ----------------------------------------------")
	log.Infoln(args.MergeFileAbsolutePath)
	log.Infoln(args.AbsoluteOutputFilepath)
	log.Infoln("---------------------------------------------------------------------------------------------------------")

	return ExecSync(&ExecArgs{
		Command:     "ffmpeg",
		CommandArgs: []string{"-y", "-hide_banner", "-loglevel", "error", "-f", "concat", "-safe", "0", "-i", args.MergeFileAbsolutePath, "-movflags", "faststart", "-codec", "copy", args.AbsoluteOutputFilepath},
		OnStart:     args.OnStart,
		OnPipeErr: func(info PipeMessage) {
			if args.OnErr != nil {
				args.OnErr(errors.New(info.Output))
			}
		},
		OnPipeOut: args.OnProgress,
	})
}

func MergeVideosReEncoded(args *MergeReEncodeArgs) error {
	log.Infoln("---------------------------------------------- Re-Encoded Merge Job ------------------------------------------")
	log.Infof("Merging %d videos with re-encoding to highest quality spec", len(args.InputFiles))
	log.Infoln("---------------------------------------------------------------------------------------------------------")

	if len(args.InputFiles) == 0 {
		return errors.New("no input files provided for merge")
	}

	// Probe all input videos to get their properties
	videoInfos := make([]*FFProbeInfo, len(args.InputFiles))
	for i, filepath := range args.InputFiles {
		video := &Video{FilePath: filepath}
		info, err := video.GetVideoInfo()
		if err != nil {
			return fmt.Errorf("error probing video '%s': %w", filepath, err)
		}
		videoInfos[i] = info
	}

	// Calculate maximum values across all videos
	maxWidth := uint(0)
	maxHeight := uint(0)
	maxFps := 0.0
	maxBitrate := uint64(0)

	for _, info := range videoInfos {
		if info.Width > maxWidth {
			maxWidth = info.Width
		}
		if info.Height > maxHeight {
			maxHeight = info.Height
		}
		if info.Fps > maxFps {
			maxFps = info.Fps
		}
		if info.BitRate > maxBitrate {
			maxBitrate = info.BitRate
		}
	}

	log.Infof("Target merge spec - Resolution: %dx%d, FPS: %.2f, Bitrate: %d", maxWidth, maxHeight, maxFps, maxBitrate)

	// Create temporary directory for re-encoded files
	tempDir := filepath.Dir(args.AbsoluteOutputFilepath)
	reEncodeExt := fmt.Sprintf("_reencode_%d", time.Now().UnixNano())

	// Re-encode all videos to the maximum spec
	reEncodedFiles := make([]string, len(args.InputFiles))
	for i, inputFile := range args.InputFiles {
		reEncodedFiles[i] = filepath.Join(tempDir, fmt.Sprintf("merge_reencode_%d%s.mp4", i, reEncodeExt))

		fpsStr := fmt.Sprintf("%.2f", maxFps)
		scaleStr := fmt.Sprintf("%d:%d", maxWidth, maxHeight)

		err := ExecSync(&ExecArgs{
			Command: "ffmpeg",
			CommandArgs: []string{
				"-y",
				"-hide_banner",
				"-loglevel", "error",
				"-i", inputFile,
				"-vf", fmt.Sprintf("scale=%s:force_original_aspect_ratio=decrease,pad=%s:(ow-iw)/2:(oh-ih)/2,format=yuv420p", scaleStr, scaleStr),
				"-r", fpsStr,
				"-c:v", "libx265",
				"-crf", "18",
				"-preset", "medium",
				"-pix_fmt", "yuv420p",
				"-movflags", "faststart",
				"-c:a", "aac",
				reEncodedFiles[i],
			},
			OnStart: args.OnStart,
			OnPipeErr: func(info PipeMessage) {
				if args.OnErr != nil {
					args.OnErr(errors.New(info.Output))
				}
			},
			OnPipeOut: args.OnProgress,
		})

		if err != nil {
			// Clean up all re-encoded files on error
			log.Errorf("Error re-encoding video '%s': %v", inputFile, err)
			for _, file := range reEncodedFiles {
				if file != "" && file != args.AbsoluteOutputFilepath {
					if errCleanup := os.Remove(file); errCleanup != nil {
						log.Warnf("Error deleting partial re-encoded file '%s': %v", file, errCleanup)
					}
				}
			}
			return fmt.Errorf("error re-encoding video '%s': %w", inputFile, err)
		}
	}

	// Create concat demuxer file
	concatFileContent := make([]string, len(reEncodedFiles))
	for i, file := range reEncodedFiles {
		concatFileContent[i] = fmt.Sprintf("file '%s'", file)
	}

	concatFilePath := filepath.Join(tempDir, fmt.Sprintf("merge_concat_%d%s.txt", time.Now().UnixNano(), reEncodeExt))
	err := os.WriteFile(concatFilePath, []byte(strings.Join(concatFileContent, "\n")), 0644)
	if err != nil {
		log.Errorf("Error writing concat file: %v", err)
		for _, file := range reEncodedFiles {
			if errCleanup := os.Remove(file); errCleanup != nil {
				log.Warnf("Error deleting re-encoded file '%s': %v", file, errCleanup)
			}
		}
		return fmt.Errorf("error writing concat demuxer file: %w", err)
	}

	// Merge re-encoded videos
	errMerge := ExecSync(&ExecArgs{
		Command: "ffmpeg",
		CommandArgs: []string{
			"-y",
			"-hide_banner",
			"-loglevel", "error",
			"-f", "concat",
			"-safe", "0",
			"-i", concatFilePath,
			"-movflags", "faststart",
			"-codec", "copy",
			args.AbsoluteOutputFilepath,
		},
		OnStart: args.OnStart,
		OnPipeErr: func(info PipeMessage) {
			if args.OnErr != nil {
				args.OnErr(errors.New(info.Output))
			}
		},
		OnPipeOut: args.OnProgress,
	})

	// Clean up re-encoded files and concat file
	for _, file := range reEncodedFiles {
		if errCleanup := os.Remove(file); errCleanup != nil {
			log.Warnf("Error deleting re-encoded file '%s': %v", file, errCleanup)
		}
	}
	if errCleanup := os.Remove(concatFilePath); errCleanup != nil {
		log.Warnf("Error deleting concat file '%s': %v", concatFilePath, errCleanup)
	}

	if errMerge != nil {
		// Clean up output file if merge failed
		if errCleanup := os.Remove(args.AbsoluteOutputFilepath); errCleanup != nil {
			log.Warnf("Error deleting failed merge output '%s': %v", args.AbsoluteOutputFilepath, errCleanup)
		}
		return fmt.Errorf("error merging re-encoded videos: %w", errMerge)
	}

	log.Infof("Successfully merged %d videos with re-encoding to %dx%d @ %.2f FPS, %d kbps", len(args.InputFiles), maxWidth, maxHeight, maxFps, maxBitrate/1000)
	return nil
}

func CutVideo(args *CuttingJob, absoluteFilepath, absoluteOutputFilepath, startIntervals, endIntervals string) error {
	log.Infoln("---------------------------------------------- Cutting Job ----------------------------------------------")
	log.Infoln(absoluteFilepath)
	log.Infoln(absoluteOutputFilepath)
	log.Infoln(startIntervals)
	log.Infoln(endIntervals)
	log.Infoln("---------------------------------------------------------------------------------------------------------")

	return ExecSync(&ExecArgs{
		Command:     "ffmpeg",
		CommandArgs: []string{"-y", "-progress", "pipe:1", "-hide_banner", "-loglevel", "error", "-i", absoluteFilepath, "-ss", startIntervals, "-to", endIntervals, "-movflags", "faststart", "-codec", "copy", absoluteOutputFilepath},
		OnStart: func(info CommandInfo) {
			args.OnStart(&info)
		},
		OnPipeErr: func(info PipeMessage) {
			log.Error(info.Output)
		},
	})
}

func ParseFFmpegKVs(text string) map[string]string {
	lines := strings.Split(text, "\n")

	kvs := make(map[string]string)
	for _, line := range lines {
		kv := strings.Split(line, "=")
		if len(kv) > 1 {
			kvs[kv[0]] = kv[1]
		}
	}

	return kvs
}

func CheckVideo(filepath string) error {
	return ExecSync(&ExecArgs{
		Command:     "ffmpeg",
		CommandArgs: []string{"-v", "error", "-i", filepath, "-f", "null", "-"},
	})
}
