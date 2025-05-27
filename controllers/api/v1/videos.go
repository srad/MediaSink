package v1

import (
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

// GenerateCovers godoc
// @Summary     Return a list of recordings
// @Description Return a list of recordings.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200
// @Failure     500 {} string "Error message"
// @Router      /videos/generate/posters [post]
func GenerateCovers(c *gin.Context) {
	appG := app.Gin{C: c}

	if err := services.GenerateVideoCovers(); err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, nil)
}

// UpdateVideoInfo godoc
// @Summary     Return a list of videos
// @Description Return a list of videos.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200
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
// @Summary     Returns if current the videos are updated.
// @Description Returns if current the videos are updated.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Success     200
// @Failure     500 {} string "Error message"
// @Router      /videos/isupdating [get]
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
		if job1, job2, err := videos.EnqueuePreviewsJob(); err != nil {
			appG.Error(http.StatusInternalServerError, err)
			return
		} else {
			appG.Response(http.StatusOK, []*database.Job{job1, job2})
		}
	}
}

// FavVideo godoc
// @Summary     Bookmark a certain video in a channel
// @Description Bookmark a certain video in a channel.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "video item id"
// @Success     200
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
// @Summary     Bookmark a certain video in a channel
// @Description Bookmark a certain video in a channel.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "video item id"
// @Success     200
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
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       limit path string int "How many videos"
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
// @Summary     Download a file from a channel
// @Description Download a file from a channel.
// @Tags        videos
// @Accept      json
// @Produce     json
// @Param       id path uint true "Recording item id"
// @Success     200
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
