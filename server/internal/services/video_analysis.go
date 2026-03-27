package services

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/server/internal/analysis/detectors"
	chaptersegments "github.com/srad/mediasink/server/internal/analysis/segments"
	"github.com/srad/mediasink/server/internal/analysis/smoothing"
	"github.com/srad/mediasink/server/internal/analysis/threshold"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/jobs"
	"github.com/srad/mediasink/server/internal/jobs/handlers"
	"github.com/srad/mediasink/server/internal/store/vector"
	"github.com/srad/mediasink/server/internal/ws"
)

func init() {
	// Register the frame analysis handler with the jobs package
	// This avoids circular imports between jobs and services packages
	jobs.RegisterAnalyzeFrameHandler(AnalyzeVideoFramesWithJob)
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
	if config == nil {
		config = detectors.DefaultDetectorConfig()
	}

	const chapterSegmenterName = "semantic_chapters"
	const highlightPipelineName = "adjacent_similarity"

	log.Infof("[AnalyzeVideoFrames] Starting analysis for recording %d with embedding=%s, chapterSegmenter=%s, highlights=%s",
		recordingID, config.SceneDetector, chapterSegmenterName, highlightPipelineName)

	if job != nil {
		handlers.EmitJobProgress(job, 0, 100, "Initializing analysis")
	}

	vectorStore := vector.Default()

	// Delete any existing analysis and stored frame vectors for this recording.
	if err := db.DeleteAnalysisByRecordingID(recordingID); err != nil {
		log.Warnf("[AnalyzeVideoFrames] Failed to delete existing analysis: %v", err)
	}
	if err := vectorStore.DeleteRecording(context.Background(), recordingID); err != nil {
		log.Warnf("[AnalyzeVideoFrames] Failed to delete existing frame vectors: %v", err)
	}

	embeddingExtractor, err := detectors.CreateEmbeddingExtractor(config.SceneDetector)
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to create embedding extractor: %v", err)
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

	recordingDuration := resolveAnalysisDuration(recordingID, frameTimestamps, job)
	frameTimestamps = clampTimestampsToDuration(frameTimestamps, recordingDuration)

	var scenes []db.SceneInfo
	var highlights []db.HighlightInfo
	var segments []db.SegmentInfo

	log.Infof("[AnalyzeVideoFrames] Using ONNX embedding extraction with semantic chapter segmentation")

	// Phase 1: run ONNX inference with no DB connection held.
	// Vectors are accumulated in memory so the database is never locked
	// during the CPU-intensive extraction step.
	embeddings := make([]vector.Embedding, 0, len(framePaths))
	chapterFrames := make([]chaptersegments.EmbeddingFrame, 0, len(framePaths))
	for i, path := range framePaths {
		frame, err := loadFrame(path)
		if err != nil {
			log.Warnf("[AnalyzeVideoFrames] Failed to load frame %s: %v", path, err)
			continue
		}

		vec, err := embeddingExtractor.ExtractFeatures(frame)
		if err != nil {
			return fmt.Errorf("feature extraction failed for frame %s: %w", path, err)
		}

		timestamp := frameTimestamps[i]
		embeddings = append(embeddings, vector.Embedding{
			Values:    vec,
			Timestamp: timestamp,
		})
		chapterFrames = append(chapterFrames, chaptersegments.EmbeddingFrame{
			Values:    vec,
			Timestamp: timestamp,
		})

		if job != nil && i%50 == 0 {
			progress := uint64(10 + int(float64(i)/float64(len(framePaths))*30))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Extracting features: %d/%d frames", i+1, len(framePaths)))
		}
	}

	if len(embeddings) < 2 {
		return fmt.Errorf("insufficient decoded frames for analysis")
	}

	// Phase 2: write all vectors in a single short transaction.
	if err := vectorStore.WriteEmbeddings(context.Background(), recordingID, embeddings); err != nil {
		return fmt.Errorf("failed to store frame vectors: %w", err)
	}
	log.Infof("[AnalyzeVideoFrames] Saved frame vectors to sqlite-vec")

	if job != nil {
		handlers.EmitJobProgress(job, 40, 100, "Features extracted, detecting chapters")
	}

	segments, err = chaptersegments.DetectChapters(chapterFrames, recordingDuration, chaptersegments.DefaultChapterConfig())
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Chapter detection failed: %v", err)
		return err
	}
	scenes = projectSegmentsToScenes(segments)

	if job != nil {
		handlers.EmitJobProgress(job, 60, 100, fmt.Sprintf("Detected %d chapters", len(segments)))
	}

	// Query consecutive cosine similarities directly from sqlite-vec for legacy highlights.
	simTimestamps, similarities, err := vectorStore.QueryConsecutiveSimilarities(context.Background(), recordingID)
	if err != nil {
		return fmt.Errorf("failed to query consecutive similarities: %w", err)
	}
	simTimestamps = clampTimestampsToDuration(simTimestamps, recordingDuration)

	highlights, err = detectHighlightsFromSimilarities(similarities, simTimestamps, job)
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Highlight detection failed: %v", err)
		return err
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

	if err := analysis.SetSegments(segments); err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to set segments: %v", err)
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

	log.Infof("[AnalyzeVideoFrames] Analysis completed (%s/%s/%s): %d segments, %d scenes, %d highlights",
		embeddingExtractor.Name(), chapterSegmenterName, highlightPipelineName, len(segments), len(scenes), len(highlights))

	ws.BroadCastClients(ws.JobDoneEvent, map[string]interface{}{
		"type":               "video_analysis",
		"recordingId":        recordingID,
		"embeddingExtractor": embeddingExtractor.Name(),
		"sceneDetector":      chapterSegmenterName,
		"highlightDetector":  highlightPipelineName,
		"segments":           len(segments),
		"scenes":             len(scenes),
		"highlights":         len(highlights),
	})

	return nil
}

// getPreviewFramePaths returns file paths and timestamps without loading images.
func getPreviewFramePaths(previewPath string) ([]string, []float64, error) {
	frames, err := LoadPreviewFrames(previewPath)
	if err != nil {
		return nil, nil, err
	}

	if len(frames) == 0 {
		return []string{}, []float64{}, nil
	}

	paths := make([]string, 0, len(frames))
	timestamps := make([]float64, 0, len(frames))
	for _, frame := range frames {
		paths = append(paths, frame.Path)
		timestamps = append(timestamps, float64(frame.Timestamp))
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

	const (
		highlightSmoothingWindow    = 5
		highlightThresholdMultiplier = 2.0
		highlightFallbackThreshold   = 0.5
		highlightMergeGapSeconds     = 6.0
		highlightMinDurationSeconds  = 8.0
		highlightMinPeakIntensity    = 0.18
	)

	smoothingMethod := smoothing.DefaultSmoothingMethod()
	smoothed := smoothingMethod.Smooth(similarities, highlightSmoothingWindow)

	thresholdMethod := threshold.NewStatisticalThresholdMethod(highlightThresholdMultiplier)
	highlightThreshold, err := thresholdMethod.Calculate(smoothed)
	if err != nil {
		log.Warnf("[Highlight Detection] Failed to calculate adaptive threshold: %v, using fallback", err)
		highlightThreshold = highlightFallbackThreshold
	}

	log.Infof("[Highlight Detection] Using %s smoothing (window=%d), threshold=%.4f via %s",
		smoothingMethod.Name(), highlightSmoothingWindow, highlightThreshold, thresholdMethod.Name())

	ranges := buildHighlightRanges(smoothed, timestamps, highlightThreshold)
	ranges = mergeHighlightRanges(ranges, highlightMergeGapSeconds)
	highlights := make([]db.HighlightInfo, 0, len(ranges))

	for i, similarity := range smoothed {
		if job != nil && i%50 == 0 {
			progress := uint64(60 + int(float64(i)/float64(len(smoothed))*20))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Highlight detection: %d/%d", i+1, len(smoothed)))
		}
		_ = similarity
	}

	for _, candidate := range ranges {
		duration := candidate.EndTime - candidate.StartTime
		if duration < highlightMinDurationSeconds || candidate.PeakIntensity < highlightMinPeakIntensity {
			continue
		}

		highlights = append(highlights, db.HighlightInfo{
			StartTime: candidate.StartTime,
			EndTime:   candidate.EndTime,
			Timestamp: candidate.PeakTimestamp,
			Intensity: clampFloat(candidate.PeakIntensity, 0, 1),
			Type:      "motion",
		})
	}

	total := len(smoothed)
	triggerRate := float64(len(highlights)) / float64(total) * 100.0
	log.Infof("[ONNX] Highlight detection: %d grouped highlights from %d pairs (threshold=%.4f, %d/%d=%.1f%% kept)",
		len(highlights), total, highlightThreshold, len(highlights), total, triggerRate)

	return highlights, nil
}

type highlightRange struct {
	StartTime     float64
	EndTime       float64
	PeakTimestamp float64
	PeakIntensity float64
}

func buildHighlightRanges(smoothed []float64, timestamps []float64, thresholdValue float64) []highlightRange {
	if len(smoothed) == 0 || len(smoothed) != len(timestamps) {
		return nil
	}

	ranges := make([]highlightRange, 0)
	active := false
	current := highlightRange{}

	for index, similarity := range smoothed {
		intensity := clampFloat(1-similarity, 0, 1)
		if similarity < thresholdValue {
			if !active {
				startTime := 0.0
				if index > 0 {
					startTime = timestamps[index-1]
				}
				current = highlightRange{
					StartTime:     startTime,
					EndTime:       timestamps[index],
					PeakTimestamp: timestamps[index],
					PeakIntensity: intensity,
				}
				active = true
			} else {
				current.EndTime = timestamps[index]
				if intensity > current.PeakIntensity {
					current.PeakIntensity = intensity
					current.PeakTimestamp = timestamps[index]
				}
			}
			continue
		}

		if active {
			ranges = append(ranges, current)
			active = false
		}
	}

	if active {
		ranges = append(ranges, current)
	}

	return ranges
}

func mergeHighlightRanges(ranges []highlightRange, mergeGap float64) []highlightRange {
	if len(ranges) <= 1 {
		return ranges
	}

	merged := []highlightRange{ranges[0]}
	for _, current := range ranges[1:] {
		lastIndex := len(merged) - 1
		last := merged[lastIndex]
		if current.StartTime-last.EndTime <= mergeGap {
			if current.EndTime > last.EndTime {
				last.EndTime = current.EndTime
			}
			if current.PeakIntensity > last.PeakIntensity {
				last.PeakIntensity = current.PeakIntensity
				last.PeakTimestamp = current.PeakTimestamp
			}
			merged[lastIndex] = last
			continue
		}
		merged = append(merged, current)
	}

	return merged
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func resolveAnalysisDuration(recordingID db.RecordingID, frameTimestamps []float64, job *db.Job) float64 {
	if job != nil && job.Recording.Duration > 0 {
		return job.Recording.Duration
	}

	recording, err := recordingID.FindRecordingByID()
	if err == nil && recording.Duration > 0 {
		return recording.Duration
	}

	if len(frameTimestamps) == 0 {
		return 0
	}

	return frameTimestamps[len(frameTimestamps)-1]
}

func clampTimestampsToDuration(timestamps []float64, duration float64) []float64 {
	if duration <= 0 || len(timestamps) == 0 {
		return timestamps
	}

	clamped := make([]float64, len(timestamps))
	for index, timestamp := range timestamps {
		if timestamp < 0 {
			clamped[index] = 0
			continue
		}
		if timestamp > duration {
			clamped[index] = duration
			continue
		}
		clamped[index] = timestamp
	}

	return clamped
}

func projectSegmentsToScenes(segments []db.SegmentInfo) []db.SceneInfo {
	if len(segments) == 0 {
		return []db.SceneInfo{}
	}

	scenes := make([]db.SceneInfo, 0, len(segments))
	for _, segment := range segments {
		scenes = append(scenes, db.SceneInfo{
			StartTime:       segment.StartTime,
			EndTime:         segment.EndTime,
			ChangeIntensity: segment.Confidence,
		})
	}
	return scenes
}

// GetAnalysisProgress returns current analysis progress for a recording.
func GetAnalysisProgress(recordingID db.RecordingID) (*db.VideoAnalysisResult, error) {
	return db.GetAnalysisByRecordingID(recordingID)
}
