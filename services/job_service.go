package services

import (
	"fmt"

	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/helpers"
	"github.com/srad/mediasink/jobs"
	"github.com/srad/mediasink/models/responses"
)

// StartJobProcessing initializes and starts all job processing workers
// This is a wrapper that maintains backward compatibility with existing code
func StartJobProcessing() {
	jobs.StartJobProcessing()
}

// StopJobProcessing stops all job processing workers
// This is a wrapper that maintains backward compatibility with existing code
func StopJobProcessing() {
	jobs.StopJobProcessing()
}

// DeleteJob marks a job as deleted and broadcasts the event
// This is a wrapper that maintains backward compatibility with existing code
func DeleteJob(id uint) error {
	return jobs.DeleteJob(id)
}

// IsJobProcessing returns whether the job processing system is active
// This is a wrapper that maintains backward compatibility with existing code
func IsJobProcessing() bool {
	return jobs.IsJobProcessing()
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
