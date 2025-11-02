package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/app"
	"github.com/srad/mediasink/database"
	"github.com/srad/mediasink/models/responses"
	"github.com/srad/mediasink/services"
)

// AnalyzeVideo godoc
// @Summary     Analyze video frames for scenes and highlights
// @Description Analyze preview frames to detect scenes and highlights. Runs in background as a job.
// @Tags        analysis
// @Accept      json
// @Produce     json
// @Param       id path uint true "recording id"
// @Success     200 {} nil
// @Failure     400 {} string "Invalid recording id"
// @Failure     500 {} string "Error message"
// @Router      /analysis/{id} [post]
func AnalyzeVideo(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	recordingID := database.RecordingID(id)

	// Get recording to verify it exists
	recording, err := recordingID.FindRecordingByID()
	if err != nil {
		appG.Error(http.StatusNotFound, err)
		return
	}

	// Create analysis job
	_, err = recording.EnqueueAnalysisJob()
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, nil)
}

// GetAnalysisResult godoc
// @Summary     Get video analysis result
// @Description Get the analysis results (scenes and highlights) for a recording
// @Tags        analysis
// @Accept      json
// @Produce     json
// @Param       id path uint true "recording id"
// @Success     200 {object} responses.AnalysisResponse
// @Failure     400 {} string "Invalid recording id"
// @Failure     404 {} string "No analysis found"
// @Failure     500 {} string "Error message"
// @Router      /analysis/{id} [get]
func GetAnalysisResult(c *gin.Context) {
	appG := app.Gin{C: c}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	recordingID := database.RecordingID(id)
	result, err := services.GetAnalysisProgress(recordingID)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	response, err := responses.NewAnalysisResponse(uint(recordingID), result)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, response)
}
