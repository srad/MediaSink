package v1

import (
	"net/http"
	"strconv"

	"github.com/srad/mediasink/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/config"
)

// GetInfo godoc
// @Summary     Get system metrics
// @Description Get system metrics
// @Tags        info
// @Accept      json
// @Produce     json
// @Param       seconds path int true "Number of seconds to measure"
// @Success     200 {object} util.SysInfo
// @Failure     500 {}  http.StatusInternalServerError
// @Router      /info/{seconds} [get]
func GetInfo(c *gin.Context) {
	appG := app.Gin{C: c}
	cfg := config.Read()

	secs := c.Param("seconds")
	val, err := strconv.ParseUint(secs, 10, 64)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	data, err := util.Info(cfg.DataDisk, cfg.NetworkDev, val)

	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, data)
}

// GetDiskInfo godoc
// @Summary     Get disk information
// @Description Get disk information
// @Tags        info
// @Accept      json
// @Produce     json
// @Success     200 {object} util.DiskInfo
// @Failure     500 {}  http.StatusInternalServerError
// @Router      /info/disk [get]
func GetDiskInfo(c *gin.Context) {
	appG := app.Gin{C: c}

	cfg := config.Read()

	info, err := util.DiskUsage(cfg.DataDisk)

	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}

	appG.Response(http.StatusOK, info)
}
