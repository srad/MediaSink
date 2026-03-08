package services

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/analysis/detectors"
	"github.com/srad/mediasink/internal/analysis/smoothing"
	"github.com/srad/mediasink/internal/analysis/threshold"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/jobs"
	"github.com/srad/mediasink/internal/jobs/handlers"
	"github.com/srad/mediasink/internal/ws"
)

func init() {
	// Register the frame analysis handler with the jobs package
	// This avoids circular imports between jobs and services packages
	jobs.RegisterAnalyzeFrameHandler(AnalyzeVideoFramesWithJob)
}

// FeatureExtractor is satisfied by ONNX detectors that expose raw inference results.
type FeatureExtractor interface {
	ExtractFeatures(frame image.Image) ([]float32, error)
}

// AnalyzeVideoFrames analyzes preview frames to detect scenes and highlights using configured detectors
func AnalyzeVideoFrames(recordingID db.RecordingID, channelName db.ChannelName) error {
	return AnalyzeVideoFramesWithConfig(recordingID, channelName, detectors.DefaultDetectorConfig(), nil)
}

// AnalyzeVideoFramesWithJob analyzes preview frames with job tracking
func AnalyzeVideoFramesWithJob(job *db.Job) error {
	return AnalyzeVideoFramesWithConfig(job.RecordingID, job.ChannelName, detectors.DefaultDetectorConfig(), job)
}

// AnalyzeVideoFramesWithConfig analyzes preview frames with specified detector configuration
func AnalyzeVideoFramesWithConfig(recordingID db.RecordingID, channelName db.ChannelName, config *detectors.DetectorConfig, job *db.Job) error {
	log.Infof("[AnalyzeVideoFrames] Starting analysis for recording %d with detectors: scene=%s, highlight=%s",
		recordingID, config.SceneDetector, config.HighlightDetector)

	if job != nil {
		handlers.EmitJobProgress(job, 0, 100, "Initializing analysis")
	}

	// Delete any existing analysis and stored frame vectors for this recording.
	if err := db.DeleteAnalysisByRecordingID(recordingID); err != nil {
		log.Warnf("[AnalyzeVideoFrames] Failed to delete existing analysis: %v", err)
	}
	if err := db.DeleteFrameVectorsByRecordingID(recordingID); err != nil {
		log.Warnf("[AnalyzeVideoFrames] Failed to delete existing frame vectors: %v", err)
	}

	sceneDetector, highlightDetector, err := detectors.CreateDetectors(config)
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to create detectors: %v", err)
		return err
	}

	previewPath := recordingID.GetPreviewFramesPath(channelName)
	log.Infof("[AnalyzeVideoFrames] Loading frame paths from %s", previewPath)

	framePaths, frameTimestamps, err := getPreviewFramePaths(previewPath)
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to get frame paths: %v", err)
		return err
	}

	if len(framePaths) < 2 {
		log.Errorf("[AnalyzeVideoFrames] Insufficient frames for analysis")
		return fmt.Errorf("insufficient frames for analysis")
	}

	log.Infof("[AnalyzeVideoFrames] Found %d frames", len(framePaths))

	if job != nil {
		handlers.EmitJobProgress(job, 10, 100, fmt.Sprintf("Found %d frames", len(framePaths)))
	}

	useOnnx := strings.Contains(sceneDetector.Name(), "onnx") || strings.Contains(highlightDetector.Name(), "onnx")

	var scenes []db.SceneInfo
	var highlights []db.HighlightInfo

	if useOnnx {
		log.Infof("[AnalyzeVideoFrames] Using ONNX streaming processing with sqlite-vec")

		var featureExtractor FeatureExtractor
		if fx, ok := sceneDetector.(FeatureExtractor); ok {
			featureExtractor = fx
		} else if fx, ok := highlightDetector.(FeatureExtractor); ok {
			featureExtractor = fx
		}
		if featureExtractor == nil {
			return fmt.Errorf("no FeatureExtractor available from ONNX detectors")
		}

		// Phase 1: run ONNX inference with no DB connection held.
		// Vectors are accumulated in memory so the database is never locked
		// during the CPU-intensive extraction step.
		type frameVec struct {
			vec       []float32
			timestamp float64
		}
		vecs := make([]frameVec, 0, len(framePaths))
		for i, path := range framePaths {
			frame, err := loadFrame(path)
			if err != nil {
				log.Warnf("[AnalyzeVideoFrames] Failed to load frame %s: %v", path, err)
				continue
			}

			vec, err := featureExtractor.ExtractFeatures(frame)
			if err != nil {
				return fmt.Errorf("feature extraction failed for frame %s: %w", path, err)
			}
			vecs = append(vecs, frameVec{vec: vec, timestamp: frameTimestamps[i]})

			if job != nil && i%50 == 0 {
				progress := uint64(10 + int(float64(i)/float64(len(framePaths))*30))
				handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Extracting features: %d/%d frames", i+1, len(framePaths)))
			}
		}

		// Phase 2: write all vectors in a single short transaction.
		if len(vecs) > 0 {
			writer, err := db.NewFrameVectorWriter(recordingID, len(vecs[0].vec))
			if err != nil {
				log.Warnf("[AnalyzeVideoFrames] Failed to create vector writer: %v", err)
			} else {
				for i, fv := range vecs {
					if err := writer.Write(fv.vec, fv.timestamp); err != nil {
						writer.Rollback()
						return fmt.Errorf("failed to write frame vector %d: %w", i, err)
					}
				}
				if err := writer.Commit(); err != nil {
					return fmt.Errorf("failed to commit frame vectors: %w", err)
				}
			}
		}
		log.Infof("[AnalyzeVideoFrames] Saved frame vectors to sqlite-vec")

		if job != nil {
			handlers.EmitJobProgress(job, 40, 100, "Features extracted, querying similarities")
		}

		// Query consecutive cosine similarities directly from sqlite-vec.
		simTimestamps, similarities, err := db.QueryConsecutiveSimilarities(recordingID)
		if err != nil {
			return fmt.Errorf("failed to query consecutive similarities: %w", err)
		}

		scenes, err = detectScenesFromSimilarities(similarities, simTimestamps, job)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Scene detection failed: %v", err)
			return err
		}

		if job != nil {
			handlers.EmitJobProgress(job, 60, 100, fmt.Sprintf("Detected %d scenes", len(scenes)))
		}

		highlights, err = detectHighlightsFromSimilarities(similarities, simTimestamps, job)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Highlight detection failed: %v", err)
			return err
		}
	} else {
		// Single-pass approach: load all frames for non-ONNX detectors (SSIM, FrameDiff).
		log.Infof("[AnalyzeVideoFrames] Loading all frames for non-ONNX detector")

		var frames []image.Image
		for i, path := range framePaths {
			frame, err := loadFrame(path)
			if err != nil {
				log.Warnf("[AnalyzeVideoFrames] Failed to load frame %s: %v", path, err)
				continue
			}
			frames = append(frames, frame)

			if job != nil && i%50 == 0 {
				progress := uint64(10 + int(float64(i)/float64(len(framePaths))*30))
				handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Loading frames: %d/%d", i+1, len(framePaths)))
			}
		}

		if job != nil {
			handlers.EmitJobProgress(job, 40, 100, fmt.Sprintf("Loaded %d frames", len(frames)))
		}

		scenes, err = sceneDetector.DetectScenes(frames, frameTimestamps)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Scene detection failed (%s): %v", sceneDetector.Name(), err)
			return err
		}

		if job != nil {
			handlers.EmitJobProgress(job, 60, 100, fmt.Sprintf("Detected %d scenes", len(scenes)))
		}

		highlights, err = highlightDetector.DetectHighlights(frames, frameTimestamps)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Highlight detection failed (%s): %v", highlightDetector.Name(), err)
			return err
		}
	}

	if job != nil {
		handlers.EmitJobProgress(job, 80, 100, fmt.Sprintf("Detected %d highlights", len(highlights)))
	}

	analysis := &db.VideoAnalysisResult{
		RecordingID: recordingID,
		Status:      db.AnalysisCompleted,
	}

	if err := analysis.SetScenes(scenes); err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to set scenes: %v", err)
		return err
	}

	if err := analysis.SetHighlights(highlights); err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to set highlights: %v", err)
		return err
	}

	if err := db.DB.Create(analysis).Error; err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to save results: %v", err)
		return err
	}

	if job != nil {
		handlers.EmitJobProgress(job, 100, 100, "Analysis complete")
	}

	log.Infof("[AnalyzeVideoFrames] Analysis completed (%s/%s): %d scenes, %d highlights",
		sceneDetector.Name(), highlightDetector.Name(), len(scenes), len(highlights))

	ws.BroadCastClients(ws.JobDoneEvent, map[string]interface{}{
		"type":              "video_analysis",
		"recordingId":       recordingID,
		"sceneDetector":     sceneDetector.Name(),
		"highlightDetector": highlightDetector.Name(),
		"scenes":            len(scenes),
		"highlights":        len(highlights),
	})

	return nil
}

// getPreviewFramePaths returns file paths and timestamps without loading images.
func getPreviewFramePaths(previewPath string) ([]string, []float64, error) {
	files, err := os.ReadDir(previewPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	type frameFile struct {
		timestamp float64
		path      string
	}

	var frameFiles []frameFile

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		var timestamp uint64
		_, err := fmt.Sscanf(file.Name(), "%d.jpg", &timestamp)
		if err != nil {
			log.Warnf("[getPreviewFramePaths] Skipping file with invalid name: %s", file.Name())
			continue
		}

		frameFiles = append(frameFiles, frameFile{
			timestamp: float64(timestamp),
			path:      filepath.Join(previewPath, file.Name()),
		})
	}

	if len(frameFiles) == 0 {
		if len(files) > 0 {
			return nil, nil, fmt.Errorf("invalid preview frame format in %s: expected files named <timestamp>.jpg", previewPath)
		}
		return []string{}, []float64{}, nil
	}

	sort.Slice(frameFiles, func(i, j int) bool {
		return frameFiles[i].timestamp < frameFiles[j].timestamp
	})

	var paths []string
	var timestamps []float64

	for _, ff := range frameFiles {
		paths = append(paths, ff.path)
		timestamps = append(timestamps, ff.timestamp)
	}

	return paths, timestamps, nil
}

// loadFrame loads a single frame from disk.
func loadFrame(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// detectScenesFromSimilarities detects scene boundaries from pre-computed cosine
// similarities (as returned by QueryConsecutiveSimilarities).
// timestamps[i] is the timestamp of the second frame in pair i.
func detectScenesFromSimilarities(similarities []float64, timestamps []float64, job *db.Job) ([]db.SceneInfo, error) {
	smoothingMethod := smoothing.DefaultSmoothingMethod()
	smoothed := smoothingMethod.Smooth(similarities, 3)

	thresholdMethod := threshold.NewStatisticalThresholdMethod(2.0)
	sceneThreshold, err := thresholdMethod.Calculate(smoothed)
	if err != nil {
		log.Warnf("[Scene Detection] Failed to calculate adaptive threshold: %v, using fallback", err)
		sceneThreshold = 0.75
	}

	log.Infof("[Scene Detection] Using %s smoothing (window=3), threshold=%.4f via %s",
		smoothingMethod.Name(), sceneThreshold, thresholdMethod.Name())

	var scenes []db.SceneInfo
	sceneStart := 0.0
	sceneChangeCount := 0

	for i, similarity := range smoothed {
		if job != nil && i%50 == 0 {
			progress := uint64(40 + int(float64(i)/float64(len(smoothed))*20))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Scene detection: %d/%d", i+1, len(smoothed)))
		}

		if similarity < sceneThreshold {
			sceneChangeCount++
			scenes = append(scenes, db.SceneInfo{
				StartTime:       sceneStart,
				EndTime:         timestamps[i],
				ChangeIntensity: 1.0 - similarity,
			})
			sceneStart = timestamps[i]
		}
	}

	// Final scene segment
	if len(timestamps) > 0 {
		scenes = append(scenes, db.SceneInfo{
			StartTime:       sceneStart,
			EndTime:         timestamps[len(timestamps)-1],
			ChangeIntensity: 0.0,
		})
	}

	total := len(similarities)
	triggerRate := float64(sceneChangeCount) / float64(total) * 100.0
	log.Infof("[ONNX] Scene detection: %d scenes from %d pairs (threshold=%.4f, %d/%d=%.1f%% triggered)",
		len(scenes), total, sceneThreshold, sceneChangeCount, total, triggerRate)

	return scenes, nil
}

// detectHighlightsFromSimilarities detects highlights from pre-computed cosine
// similarities (as returned by QueryConsecutiveSimilarities).
// timestamps[i] is the timestamp of the second frame in pair i.
func detectHighlightsFromSimilarities(similarities []float64, timestamps []float64, job *db.Job) ([]db.HighlightInfo, error) {
	if len(similarities) < 2 {
		return nil, nil
	}

	smoothingMethod := smoothing.DefaultSmoothingMethod()
	smoothed := smoothingMethod.Smooth(similarities, 3)

	thresholdMethod := threshold.NewStatisticalThresholdMethod(2.5)
	highlightThreshold, err := thresholdMethod.Calculate(smoothed)
	if err != nil {
		log.Warnf("[Highlight Detection] Failed to calculate adaptive threshold: %v, using fallback", err)
		highlightThreshold = 0.62
	}

	log.Infof("[Highlight Detection] Using %s smoothing (window=3), threshold=%.4f via %s",
		smoothingMethod.Name(), highlightThreshold, thresholdMethod.Name())

	var highlights []db.HighlightInfo
	highlightCount := 0

	for i, similarity := range smoothed {
		if job != nil && i%50 == 0 {
			progress := uint64(60 + int(float64(i)/float64(len(smoothed))*20))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Highlight detection: %d/%d", i+1, len(smoothed)))
		}

		if similarity < highlightThreshold {
			highlightCount++
			highlights = append(highlights, db.HighlightInfo{
				Timestamp: timestamps[i],
				Intensity: 1.0 - similarity,
				Type:      "motion",
			})
		}
	}

	total := len(smoothed)
	triggerRate := float64(highlightCount) / float64(total) * 100.0
	log.Infof("[ONNX] Highlight detection: %d highlights from %d pairs (threshold=%.4f, %d/%d=%.1f%% triggered)",
		len(highlights), total, highlightThreshold, highlightCount, total, triggerRate)

	return highlights, nil
}

// GetAnalysisProgress returns current analysis progress for a recording.
func GetAnalysisProgress(recordingID db.RecordingID) (*db.VideoAnalysisResult, error) {
	return db.GetAnalysisByRecordingID(recordingID)
}
