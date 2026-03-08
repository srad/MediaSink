package v1

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/models/responses"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/internal/services"
)

// IsRecording godoc
// @Summary     Get recorder status
// @Description Get the current recording/streaming recorder status
// @Tags        recorder
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.RecordingStatusResponse
// @Failure     500 {} string "Error message"
// @Router      /recorder [get]
func IsRecording(c *gin.Context) {
	appG := app.Gin{C: c}
	appG.Response(http.StatusOK, &responses.RecordingStatusResponse{IsRecording: services.IsRecorderActive()})
}

// StopRecorder godoc
// @Summary     Pause the recorder
// @Description Stop/pause the recording and streaming recorder
// @Tags        recorder
// @Accept      json
// @Produce     json
// @Success     200 {} nil
// @Failure     500 {} string "Error message"
// @Router      /recorder/pause [post]
func StopRecorder(c *gin.Context) {
	appG := app.Gin{C: c}

	go services.StopRecorder()

	appG.Response(http.StatusOK, nil)
}

// StartRecorder godoc
// @Summary     Resume the recorder
// @Description Resume/restart the recording and streaming recorder
// @Tags        recorder
// @Accept      json
// @Produce     json
// @Success     200 {} nil
// @Failure     500 {} string "Error message"
// @Router      /recorder/resume [post]
func StartRecorder(c *gin.Context) {
	appG := app.Gin{C: c}

	log.Infoln("Resuming recorder")
	services.StartRecorder()
	appG.Response(http.StatusOK, nil)
}
