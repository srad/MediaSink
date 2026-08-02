package v1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/srad/mediasink/server/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/server/internal/app"
	"github.com/srad/mediasink/server/config"
)

// maxInfoSeconds caps the measurement window. util.Info sleeps once in
// CPUUsage and again in NetMeasure, so a request occupies a goroutine for
// roughly 2x this value. Clients ask for 1 second.
const maxInfoSeconds = 60

// GetInfo godoc
// @Summary     Get system metrics
// @Description Get system metrics. The measurement window is capped at 60 seconds.
// @Tags        info
// @Accept      json
// @Produce     json
// @Param       seconds path int true "Number of seconds to measure (1-60)"
// @Success     200 {object} util.SysInfo
// @Failure     400 {}  http.StatusBadRequest
// @Failure     500 {}  http.StatusInternalServerError
// @Router      /info/{seconds} [get]
func GetInfo(c *gin.Context) {
	appG := app.Gin{C: c}

	// Validate before reading config or measuring anything: util.Info sleeps
	// for the requested duration, so a rejected request must cost nothing.
	secs := c.Param("seconds")
	val, err := strconv.ParseUint(secs, 10, 64)
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	if val == 0 || val > maxInfoSeconds {
		appG.Error(http.StatusBadRequest, fmt.Errorf("seconds must be between 1 and %d", maxInfoSeconds))
		return
	}

	cfg := config.Read()
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
