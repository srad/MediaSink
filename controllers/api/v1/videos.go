package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/srad/mediasink/helpers"
	"github.com/srad/mediasink/models/requests"
	"github.com/srad/mediasink/models/responses"
	"github.com/srad/mediasink/queries"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/app"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/services"
)

// GetVideos godoc
// @Summary     Return a list of videos
// @Description Return a list of videos.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200 {object} []database.Recording
// @Failure     500 {} string "Error message"
// @Router      /videos [get]
func GetVideos(c *gin.Context) {
	appG := app.Gin{C: c}
	videos, err := database.RecordingsList()

	if err != nil {
		appG.Error(http.StatusInternalServerError, nil)
		return
	}

	appG.Response(http.StatusOK, videos)
}

// UpdateVideoInfo godoc
// @Summary     Update video metadata information
// @Description Update metadata information for all videos in the system
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200 {} nil
// @Failure     500 {} string "Error message"
// @Router      /videos/updateinfo [post]
func UpdateVideoInfo(c *gin.Context) {
	appG := app.Gin{C: c}
	// TODO Make into a cancelable job
	if err := services.UpdateVideoInfo(); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}
	appG.Response(http.StatusOK, nil)
}

// IsUpdatingVideoInfo godoc
// @Summary     Check if video metadata update is in progress
// @Description Get the status of the video metadata update process
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200 {object} bool
// @Failure     500 {} string "Error message"
// @Router      /videos/isupdating [post]
func IsUpdatingVideoInfo(c *gin.Context) {
	appG := app.Gin{C: c}
	// TODO: do it
	appG.Response(http.StatusOK, services.IsUpdatingRecordings())
}

// GetVideo godoc
// @Summary     Return a list of videos for a particular channel
// @Description Return a list of videos for a particular channel.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "videos item id"
// @Success     200 {object} database.Recording
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id} [get]
func GetVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	video, err := database.RecordingID(id).FindRecordingByID()
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, &video)
}

// GetBookmarkedVideos godoc
// @Summary     Returns all bookmarked videos.
// @Description Returns all bookmarked videos.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200 {object} []database.Recording
// @Failure     500 {} string "Error message"
// @Router      /videos/bookmarks [get]
func GetBookmarkedVideos(c *gin.Context) {
	appG := app.Gin{C: c}
	videos, err := database.BookmarkList()

	if err != nil {
		appG.Error(http.StatusInternalServerError, nil)
		return
	}

	appG.Response(http.StatusOK, videos)
}

// GenerateVideoPreviews godoc
// @Summary     Generate preview for a certain video in a channel
// @Description Generate preview for a certain video in a channel.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "videos item id"
// @Success     200 {object} []database.Job
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/preview [post]
func GenerateVideoPreviews(c *gin.Context) {
	appG := app.Gin{C: c}

	id, errConvert := strconv.ParseUint(c.Param("id"), 10, 32)
	if errConvert != nil {
		appG.Error(http.StatusBadRequest, errConvert)
		return
	}

	if videos, err := database.RecordingID(id).FindRecordingByID(); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	} else {
		if job, err := videos.EnqueuePreviewFramesJob(); err != nil {
			appG.Error(http.StatusInternalServerError, err)
			return
		} else {
			appG.Response(http.StatusOK, job)
		}
	}
}

// FavVideo godoc
// @Summary     Bookmark a video
// @Description Bookmark/favorite a video for easy access
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "video item id"
// @Success     200 {} nil
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/fav [patch]
func FavVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	if err := database.FavRecording(uint(id), true); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, nil)
}

// UnfavVideo godoc
// @Summary     Remove video from bookmarks
// @Description Remove/unbookmark a video from favorites
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "video item id"
// @Success     200 {} nil
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/unfav [patch]
func UnfavVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	if err := database.FavRecording(uint(id), false); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, nil)
}

// CutVideo godoc
// @Summary     Cut a video and merge all defined segments
// @Description Cut a video and merge all defined segments
// @Tags        videos
// @Param       id path uint true "video item id"
// @Param       CutRequest body requests.CutRequest true "Start and end timestamp of cutting sequences."
// @Accept      json
// @Produce     json
// @Success     200 {object} database.Job
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/cut [post]
func CutVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	cutRequest := &requests.CutRequest{}
	if err := c.BindJSON(cutRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	args := &helpers.CutArgs{
		Starts:                cutRequest.Starts,
		Ends:                  cutRequest.Ends,
		DeleteAfterCompletion: cutRequest.DeleteAfterCompletion,
	}
	if job, err := database.EnqueueCuttingJob(uint(id), args); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	} else {
		appG.Response(http.StatusOK, job)
	}
}

// ConvertVideo godoc
// @Summary     Cut a video and merge all defined segments
// @Description Cut a video and merge all defined segments
// @Tags        videos
// @Param       id path uint true "video item id"
// @Param       mediaType path string true "Media type to convert to: 720, 1080, mp3"
// @Accept      json
// @Produce     json
// @Success     200 {object} database.Job
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/{mediaType}/convert [post]
func ConvertVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}
	mediaType := c.Param("mediaType")

	if video, err := database.RecordingID(id).FindRecordingByID(); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	} else {
		if job, err := video.EnqueueConversionJob(mediaType); err != nil {
			appG.Error(http.StatusInternalServerError, err)
			return
		} else {
			appG.Response(http.StatusOK, job)
		}
	}
}

// FilterVideos godoc
// @Summary     Get the top N the latest videos.
// @Description Get the top N the latest videos.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       VideoFilterRequest body requests.VideoFilterRequest true "Video filter containing column name, sort order and skip and limit"
// @Success     200 {object} responses.VideoFilterResponse
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/filter [post]
func FilterVideos(c *gin.Context) {
	appG := app.Gin{C: c}

	var request requests.VideoFilterRequest

	if err := c.BindJSON(&request); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Validation
	if !request.SortOrder.IsValid() {
		request.SortOrder = queries.SortDesc
	}

	if !request.SortColumn.IsValid() {
		request.SortColumn = requests.SortColumnCreatedAt
	}

	videos, totalCount, err := database.SortBy(request.SortColumn.String(), request.SortOrder.String(), request.Skip, request.Take)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, responses.VideoFilterResponse{
		Videos:     videos,
		TotalCount: totalCount,
		Skip:       request.Skip,
		Take:       request.Take,
	})
}

// GetRandomVideos godoc
// @Summary     Get random videos
// @Description Get a random selection of videos from the system
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       limit path int true "Number of random videos to return"
// @Success     200 {object} []database.Recording
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/random/{limit} [get]
func GetRandomVideos(c *gin.Context) {
	appG := app.Gin{C: c}

	limit, err := strconv.Atoi(c.Param("limit"))
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	videos, err := database.FindRandom(limit)

	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, videos)
}

// DownloadVideo godoc
// @Summary     Download a video file
// @Description Download a video file as an attachment
// @Tags        videos
// @Accept      json
// @Produce     octet-stream
// @Param       id path uint true "Recording item id"
// @Success     200 {file} file "Video file"
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/download [get]
func DownloadVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	if video, err := database.FindRecordingByID(database.RecordingID(id)); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	} else {
		c.FileAttachment(video.AbsoluteChannelFilepath(), video.Filename.String())
	}
}

// DeleteVideo godoc
// @Summary     Delete video
// @Description Delete video
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "video item id"
// @Success     200
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id} [delete]
func DeleteVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	video, err := database.RecordingID(id).FindRecordingByID()
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	if video != nil {
		if err2 := video.DestroyRecording(); err2 != nil {
			appG.Error(http.StatusInternalServerError, err2)
			return
		}
	}

	appG.Response(http.StatusOK, nil)
}

// MergeVideos godoc
// @Summary     Merge multiple videos
// @Description Merge multiple videos with optional re-encoding to highest quality spec
// @Tags        channels
// @Param       id path uint true "Channel id"
// @Param       MergeRequest body requests.MergeRequest true "Recording IDs and merge options"
// @Accept      json
// @Produce     json
// @Success     200 {object} database.Job
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /channels/{id}/merge [post]
func MergeVideos(c *gin.Context) {
	appG := app.Gin{C: c}

	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	mergeRequest := &requests.MergeRequest{}
	if err := c.BindJSON(&mergeRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Validate request
	if err := appG.ValidateRequest(mergeRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	job, err := database.EnqueueMergeJob(database.ChannelID(channelID), mergeRequest.RecordingIDs, mergeRequest.ReEncode)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, job)
}

// EnhanceVideo godoc
// @Summary     Enhance video quality
// @Description Enhance a video with denoising, upscaling, and sharpening
// @Tags        videos
// @Param       id path uint true "Recording id"
// @Param       EnhanceRequest body requests.EnhanceRequest true "Enhancement parameters"
// @Accept      json
// @Produce     json
// @Success     200 {object} database.Job
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/enhance [post]
func EnhanceVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	enhanceRequest := &requests.EnhanceRequest{}
	if err := c.BindJSON(&enhanceRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Validate request
	if err := appG.ValidateRequest(enhanceRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Convert to helpers types
	targetRes := helpers.ResolutionType(enhanceRequest.TargetResolution)
	encodingPreset := helpers.EncodingPreset(enhanceRequest.EncodingPreset)

	// Get CRF value (use provided or default to 18)
	crf := uint(18)
	if enhanceRequest.CRF != nil {
		crf = *enhanceRequest.CRF
	}

	// Create enhance args
	enhanceArgs := &helpers.EnhanceArgs{
		TargetResolution: targetRes,
		DenoiseStrength:  enhanceRequest.DenoiseStrength,
		SharpenStrength:  enhanceRequest.SharpenStrength,
		ApplyNormalize:   enhanceRequest.ApplyNormalize,
		EncodingPreset:   encodingPreset,
		CRF:              crf,
	}

	job, err := database.EnqueueEnhanceVideoJob(uint(id), enhanceArgs)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, job)
}

// EstimateEnhancement godoc
// @Summary     Estimate video enhancement file size
// @Description Estimate the output file size for video enhancement with given parameters
// @Tags        videos
// @Param       id path uint true "Recording id"
// @Param       EstimateEnhancementRequest body requests.EstimateEnhancementRequest true "Enhancement parameters"
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.EstimateEnhancementResponse
// @Failure     400 {} string "Error message"
// @Failure     500 {} string "Error message"
// @Router      /videos/{id}/estimate-enhancement [post]
func EstimateEnhancement(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	estimateRequest := &requests.EstimateEnhancementRequest{}
	if err := c.BindJSON(&estimateRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Validate request
	if err := appG.ValidateRequest(estimateRequest); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	// Get recording
	recording, err := database.RecordingID(id).FindRecordingByID()
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	if recording == nil {
		appG.Error(http.StatusBadRequest, fmt.Errorf("recording not found"))
		return
	}

	// Convert to helpers types
	targetRes := helpers.ResolutionType(estimateRequest.TargetResolution)

	// Get CRF value (use provided or default to 18)
	crf := uint(18)
	if estimateRequest.CRF != nil {
		crf = *estimateRequest.CRF
	}

	// Estimate file size
	estimatedSize, err := services.EstimateEnhancementFileSize(recording, targetRes, crf)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	// Calculate compression ratio
	compressionRatio := 1.0
	if recording.Size > 0 {
		compressionRatio = float64(estimatedSize) / float64(recording.Size)
	}

	response := &responses.EstimateEnhancementResponse{
		InputFileSize:     int64(recording.Size),
		EstimatedFileSize: estimatedSize,
		EstimatedFileSizeM: float64(estimatedSize) / (1024 * 1024),
		CompressionRatio:  compressionRatio,
	}

	appG.Response(http.StatusOK, response)
}

// GetEnhancementDescriptions godoc
// @Summary     Get enhancement parameter descriptions
// @Description Return descriptions for all video enhancement parameters (presets, CRF values, resolutions, filters)
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.EnhancementDescriptions
// @Failure     500 {} string "Error message"
// @Router      /videos/enhance/descriptions [get]
func GetEnhancementDescriptions(c *gin.Context) {
	appG := app.Gin{C: c}

	descriptions := services.GetEnhancementDescriptions()
	appG.Response(http.StatusOK, descriptions)
}
