package v1

import (
	"net/http"

	"github.com/srad/mediasink/internal/models/responses"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/internal/services"
)

// TriggerImport godoc
// @Summary     Run once the import of mp4 files in the recordings folder
// @Schemes
// @Description Import all mp4 files in the recordings directory that are not yet in the system database
// @Tags        admin
// @Accept      json
// @Produce     json
// @Success     200 {} nil
// @Failure     500 {} http.StatusInternalServerError
// @Router      /admin/import [post]
func TriggerImport(c *gin.Context) {
	appG := app.Gin{C: c}

	services.StopImport()
	services.StartImport()

	appG.Response(http.StatusOK, nil)
}

// GetImportInfo godoc
// @Summary     Returns current import progress information
// @Schemes
// @Description Get the current import progress status and information
// @Tags        admin
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.ImportInfoResponse
// @Failure     500 {} http.StatusInternalServerError
// @Router      /admin/import [get]
func GetImportInfo(c *gin.Context) {
	appG := app.Gin{C: c}

	progress, size := services.GetImportProgress()

	info := responses.ImportInfoResponse{
		IsImporting: services.IsImporting(),
		Progress:    progress,
		Size:        size,
	}

	appG.Response(http.StatusOK, info)
}

// RegenerateChapters godoc
// @Summary     Remove stale chapter jobs and enqueue fresh analysis for all recordings
// @Schemes
// @Description Deletes existing analyze-frames jobs and creates a fresh chapter-analysis job for every recording
// @Tags        admin
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.RegenerateChaptersResponse
// @Failure     500 {} http.StatusInternalServerError
// @Router      /admin/chapters/regenerate [post]
func RegenerateChapters(c *gin.Context) {
	appG := app.Gin{C: c}

	result, err := services.RegenerateAllChapters()
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, result)
}

// GetVersion godoc
// @Summary     Returns server version information
// @Schemes
// @Description version information
// @Tags        admin
// @Accept      json
// @Produce     json
// @Success     200 {object} responses.ServerInfoResponse
// @Failure     500 {} http.StatusInternalServerError
// @Router      /admin/version [get]
func GetVersion(version, commit string) func(c *gin.Context) {
	return func(c *gin.Context) {
		appG := app.Gin{C: c}

		appG.Response(http.StatusOK, responses.ServerInfoResponse{Commit: commit, Version: version})
	}
}
