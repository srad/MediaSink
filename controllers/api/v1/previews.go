package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/app"
	"github.com/srad/mediasink/services"
)

// RegenerateAllPreviews godoc
// @Summary     Regenerate all preview frames
// @Description Delete and regenerate preview frames for all recordings. Runs in background and provides progress updates via WebSocket.
// @Tags        previews
// @Accept      json
// @Produce     json
// @Success     200 {} nil
// @Failure     409 {} string "Regeneration already in progress"
// @Failure     500 {} string "Error message"
// @Router      /previews/regenerate [post]
func RegenerateAllPreviews(c *gin.Context) {
	appG := app.Gin{C: c}

	if err := services.RegenerateAllPreviews(); err != nil {
		appG.Error(http.StatusConflict, err)
		return
	}

	appG.Response(http.StatusOK, nil)
}

// GetRegenerationProgress godoc
// @Summary     Get preview regeneration progress
// @Description Get the current progress of preview frame regeneration
// @Tags        previews
// @Accept      json
// @Produce     json
// @Success     200 {object} services.RegenerationProgress
// @Failure     500 {} string "Error message"
// @Router      /previews/regenerate [get]
func GetRegenerationProgress(c *gin.Context) {
	appG := app.Gin{C: c}

	progress := services.GetRegenerationProgress()
	appG.Response(http.StatusOK, progress)
}
