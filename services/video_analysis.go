package services

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/disintegration/imaging"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/analysis/detectors"
	"github.com/srad/mediasink/analysis/threshold"
	"github.com/srad/mediasink/analysis/smoothing"
	"github.com/srad/mediasink/analysis/metrics"
	"github.com/srad/mediasink/jobs"
	"github.com/srad/mediasink/jobs/handlers"
	"github.com/srad/mediasink/network"
	"gonum.org/v1/gonum/mat"
)

func init() {
	// Register the frame analysis handler with the jobs package
	// This avoids circular imports between jobs and services packages
	jobs.RegisterAnalyzeFrameHandler(AnalyzeVideoFramesWithJob)
}

type FeatureExtractor interface {
	ExtractFeatures(frame image.Image) (*mat.VecDense, error)
}

// AnalyzeVideoFrames analyzes preview frames to detect scenes and highlights using configured detectors
func AnalyzeVideoFrames(recordingID database.RecordingID, channelName database.ChannelName) error {
	return AnalyzeVideoFramesWithConfig(recordingID, channelName, detectors.DefaultDetectorConfig(), nil)
}

// AnalyzeVideoFramesWithJob analyzes preview frames with job tracking
func AnalyzeVideoFramesWithJob(job *database.Job) error {
	return AnalyzeVideoFramesWithConfig(job.RecordingID, job.ChannelName, detectors.DefaultDetectorConfig(), job)
}

// AnalyzeVideoFramesWithConfig analyzes preview frames with specified detector configuration
func AnalyzeVideoFramesWithConfig(recordingID database.RecordingID, channelName database.ChannelName, config *detectors.DetectorConfig, job *database.Job) error {
	log.Infof("[AnalyzeVideoFrames] Starting analysis for recording %d with detectors: scene=%s, highlight=%s",
		recordingID, config.SceneDetector, config.HighlightDetector)

	// Emit progress: Starting (0%)
	if job != nil {
		handlers.EmitJobProgress(job, 0, 100, "Initializing analysis")
	}

	// Delete any existing analysis for this recording (ensure only one per recording)
	if err := database.DeleteAnalysisByRecordingID(recordingID); err != nil {
		log.Warnf("[AnalyzeVideoFrames] Failed to delete existing analysis: %v", err)
		// Log warning but continue - might not have existing analysis
	}

	// Create detectors based on configuration
	sceneDetector, highlightDetector, err := detectors.CreateDetectors(config)
	if err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to create detectors: %v", err)
		return err
	}

	// Get preview frames directory
	previewPath := recordingID.GetPreviewFramesPath(channelName)
	log.Infof("[AnalyzeVideoFrames] Loading frame paths from %s", previewPath)

	// Get frame paths and timestamps (don't load images yet)
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

	// Emit progress: Frame paths loaded (10%)
	if job != nil {
		handlers.EmitJobProgress(job, 10, 100, fmt.Sprintf("Found %d frames", len(framePaths)))
	}

	// Check if we're using TensorFlow detectors (need two-pass streaming)
	useTensorFlow := strings.Contains(sceneDetector.Name(), "tensorflow") || strings.Contains(highlightDetector.Name(), "tensorflow")

	var scenes []database.SceneInfo
	var highlights []database.HighlightInfo

	if useTensorFlow {
		// Two-pass approach: Extract features first, then run detection
		log.Infof("[AnalyzeVideoFrames] Using two-pass TensorFlow processing")

		var featureExtractor FeatureExtractor
		if strings.Contains(sceneDetector.Name(), "tensorflow") {
			featureExtractor = sceneDetector.(FeatureExtractor)
		} else if strings.Contains(highlightDetector.Name(), "tensorflow") {
			featureExtractor = highlightDetector.(FeatureExtractor)
		}

		// Pass 1: Extract all features (stream through frames once)
		features, err := extractFeaturesStreaming(framePaths, job, featureExtractor)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Feature extraction failed: %v", err)
			return err
		}

		// Emit progress: Features extracted (40%)
		if job != nil {
			handlers.EmitJobProgress(job, 40, 100, fmt.Sprintf("Extracted features from %d frames", len(features)))
		}

		// Pass 2: Run detection on features
		scenes, err = detectScenesFromFeatures(sceneDetector, features, frameTimestamps, job)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Scene detection failed (%s): %v", sceneDetector.Name(), err)
			return err
		}

		// Emit progress: Scene detection complete (60%)
		if job != nil {
			handlers.EmitJobProgress(job, 60, 100, fmt.Sprintf("Detected %d scenes", len(scenes)))
		}

		highlights, err = detectHighlightsFromFeatures(highlightDetector, features, frameTimestamps, job)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Highlight detection failed (%s): %v", highlightDetector.Name(), err)
			return err
		}
	} else {
		// Single-pass approach: Load all frames for non-TensorFlow detectors
		log.Infof("[AnalyzeVideoFrames] Loading all frames for non-TensorFlow detector")

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

		// Emit progress: Frames loaded (40%)
		if job != nil {
			handlers.EmitJobProgress(job, 40, 100, fmt.Sprintf("Loaded %d frames", len(frames)))
		}

		scenes, err = sceneDetector.DetectScenes(frames, frameTimestamps)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Scene detection failed (%s): %v", sceneDetector.Name(), err)
			return err
		}

		// Emit progress: Scene detection complete (60%)
		if job != nil {
			handlers.EmitJobProgress(job, 60, 100, fmt.Sprintf("Detected %d scenes", len(scenes)))
		}

		highlights, err = highlightDetector.DetectHighlights(frames, frameTimestamps)
		if err != nil {
			log.Errorf("[AnalyzeVideoFrames] Highlight detection failed (%s): %v", highlightDetector.Name(), err)
			return err
		}
	}

	// Emit progress: Highlight detection complete (80%)
	if job != nil {
		handlers.EmitJobProgress(job, 80, 100, fmt.Sprintf("Detected %d highlights", len(highlights)))
	}

	// Create analysis record only after successful detection
	analysis := &database.VideoAnalysisResult{
		RecordingID: recordingID,
		Status:      database.AnalysisCompleted,
	}

	// Set results
	if err := analysis.SetScenes(scenes); err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to set scenes: %v", err)
		return err
	}

	if err := analysis.SetHighlights(highlights); err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to set highlights: %v", err)
		return err
	}

	// Save results to database
	if err := database.DB.Create(analysis).Error; err != nil {
		log.Errorf("[AnalyzeVideoFrames] Failed to save results: %v", err)
		return err
	}

	// Emit progress: Results saved (100%)
	if job != nil {
		handlers.EmitJobProgress(job, 100, 100, "Analysis complete")
	}

	log.Infof("[AnalyzeVideoFrames] Analysis completed (%s/%s): %d scenes, %d highlights",
		sceneDetector.Name(), highlightDetector.Name(), len(scenes), len(highlights))

	// Broadcast completion
	network.BroadCastClients(network.JobDoneEvent, map[string]interface{}{
		"type":            "video_analysis",
		"recordingId":     recordingID,
		"sceneDetector":   sceneDetector.Name(),
		"highlightDetector": highlightDetector.Name(),
		"scenes":          len(scenes),
		"highlights":      len(highlights),
	})

	return nil
}

// getPreviewFramePaths returns file paths and timestamps without loading images
func getPreviewFramePaths(previewPath string) ([]string, []float64, error) {
	files, err := os.ReadDir(previewPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Map filename (timestamp) to file info for sorting
	type frameFile struct {
		timestamp float64
		path      string
	}

	var frameFiles []frameFile

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Parse timestamp from filename (e.g., "0.jpg", "10.jpg", "100.jpg")
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

	// Sort by timestamp
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

// loadFrame loads a single frame from disk
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

// extractFeaturesStreaming extracts TensorFlow features from all frames (Pass 1)
func extractFeaturesStreaming(framePaths []string, job *database.Job, featureExtractor FeatureExtractor) ([]*mat.VecDense, error) {
	var features []*mat.VecDense

	for i, path := range framePaths {
		// Load frame
		frame, err := loadFrame(path)
		if err != nil {
			log.Warnf("[extractFeaturesStreaming] Failed to load frame %s: %v", path, err)
			continue
		}

		// Resize frame if using MobileViT
		if featureExtractor != nil {
			switch d := featureExtractor.(type) {
			case detectors.SceneDetector:
				if strings.Contains(d.Name(), "mobilevit") {
					frame = imaging.Resize(frame, 256, 256, imaging.Lanczos)
				}
			case detectors.HighlightDetector:
				if strings.Contains(d.Name(), "mobilevit") {
					frame = imaging.Resize(frame, 256, 256, imaging.Lanczos)
				}
			}
		}

		// Extract features using TensorFlow
		featureVector, err := featureExtractor.ExtractFeatures(frame)
		if err != nil {
			return nil, fmt.Errorf("feature extraction failed for frame %s: %w", path, err)
		}

		features = append(features, featureVector)

		// Emit progress during feature extraction (10% -> 40%)
		if job != nil && i%50 == 0 {
			progress := uint64(10 + int(float64(i)/float64(len(framePaths))*30))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Extracting features: %d/%d frames", i+1, len(framePaths)))
		}
	}

	return features, nil
}


// detectScenesFromFeatures detects scenes using pre-extracted features (Pass 2)
func detectScenesFromFeatures(detector detectors.SceneDetector, features []*mat.VecDense, timestamps []float64, job *database.Job) ([]database.SceneInfo, error) {
	// First pass: collect all similarity scores
	var similarities []float64
	for i := 1; i < len(features); i++ {
		similarity := metrics.CosineSimilarity(features[i-1], features[i])
		similarities = append(similarities, similarity)
	}

	// Apply temporal smoothing with 3-frame window using median smoothing (best edge preservation)
	smoothingMethod := smoothing.DefaultSmoothingMethod()
	smoothedSimilarities := smoothingMethod.Smooth(similarities, 3)

	// Calculate adaptive threshold using statistical method
	thresholdMethod := threshold.NewStatisticalThresholdMethod(2.0) // k=2.0 for more sensitive scene detection
	sceneThreshold, err := thresholdMethod.Calculate(smoothedSimilarities)
	if err != nil {
		log.Warnf("[Scene Detection] Failed to calculate adaptive threshold: %v, using fallback", err)
		sceneThreshold = 0.75 // Fallback threshold
	}

	log.Infof("[Scene Detection] Using %s smoothing with window size 3", smoothingMethod.Name())

	// Second pass: detect scenes using calculated threshold
	var scenes []database.SceneInfo
	sceneStart := 0.0
	sceneChangeCount := 0

	for i := 0; i < len(smoothedSimilarities); i++ {
		similarity := smoothedSimilarities[i]
		intensity := 1.0 - similarity

		// Emit progress during scene detection (40% -> 60%)
		if job != nil && i%50 == 0 {
			progress := uint64(40 + int(float64(i)/float64(len(features))*20))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Scene detection: %d/%d", i+1, len(features)))
		}

		// Scene change detected
		if similarity < sceneThreshold {
			sceneChangeCount++
			scenes = append(scenes, database.SceneInfo{
				StartTime:       sceneStart,
				EndTime:         timestamps[i+1],
				ChangeIntensity: intensity,
			})
			sceneStart = timestamps[i+1]
		}
	}

	// Add final scene
	if len(features) > 0 {
		scenes = append(scenes, database.SceneInfo{
			StartTime:       sceneStart,
			EndTime:         timestamps[len(timestamps)-1],
			ChangeIntensity: 0.0,
		})
	}

	totalComparisons := len(similarities)
	triggerRate := float64(sceneChangeCount) / float64(totalComparisons) * 100.0
	log.Infof("[TensorFlow] Scene detection: Detected %d scenes from %d frames (adaptive threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		len(scenes), len(features), sceneThreshold, thresholdMethod.Name(), sceneChangeCount, totalComparisons, triggerRate)

	return scenes, nil
}

// detectHighlightsFromFeatures detects highlights using pre-extracted features (Pass 2)
func detectHighlightsFromFeatures(detector detectors.HighlightDetector, features []*mat.VecDense, timestamps []float64, job *database.Job) ([]database.HighlightInfo, error) {
	if len(features) < 2 {
		return nil, nil
	}

	// First pass: collect all similarity scores
	var similarities []float64
	for i := 1; i < len(features); i++ {
		similarity := metrics.CosineSimilarity(features[i-1], features[i])
		similarities = append(similarities, similarity)
	}

	// Apply temporal smoothing with 3-frame window using median smoothing (best edge preservation)
	smoothingMethod := smoothing.DefaultSmoothingMethod()
	smoothedSimilarities := smoothingMethod.Smooth(similarities, 3)

	// Calculate adaptive threshold using statistical method
	thresholdMethod := threshold.NewStatisticalThresholdMethod(2.5) // k=2.5 for highlight detection
	highlightThreshold, err := thresholdMethod.Calculate(smoothedSimilarities)
	if err != nil {
		log.Warnf("[Highlight Detection] Failed to calculate adaptive threshold: %v, using fallback", err)
		highlightThreshold = 0.62 // Fallback threshold
	}

	log.Infof("[Highlight Detection] Using %s smoothing with window size 3", smoothingMethod.Name())

	// Second pass: detect highlights using calculated threshold
	var highlights []database.HighlightInfo
	highlightCount := 0

	for i := 0; i < len(smoothedSimilarities); i++ {
		similarity := smoothedSimilarities[i]

		// Emit progress during highlight detection (60% -> 80%)
		if job != nil && i%50 == 0 {
			progress := uint64(60 + int(float64(i)/float64(len(features))*20))
			handlers.EmitJobProgress(job, progress, 100, fmt.Sprintf("Highlight detection: %d/%d", i+1, len(features)))
		}

		// Frame is significantly different from the previous one (highlight)
		if similarity < highlightThreshold {
			highlightCount++
			highlights = append(highlights, database.HighlightInfo{
				Timestamp: timestamps[i+1],
				Intensity: 1.0 - similarity,
				Type:      "motion",
			})
		}
	}

	triggerRate := float64(highlightCount) / float64(len(smoothedSimilarities)) * 100.0
	log.Infof("[TensorFlow] Highlight detection: Detected %d highlights from %d frames (adaptive threshold=%.4f via %s, %d/%d=%.1f%% triggered)",
		len(highlights), len(features), highlightThreshold, thresholdMethod.Name(), highlightCount, len(smoothedSimilarities), triggerRate)

	return highlights, nil
}


// GetAnalysisProgress returns current analysis progress for a recording
func GetAnalysisProgress(recordingID database.RecordingID) (*database.VideoAnalysisResult, error) {
	return database.GetAnalysisByRecordingID(recordingID)
}
