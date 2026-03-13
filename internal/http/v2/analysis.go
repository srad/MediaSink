package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/models/responses"
	analysissvc "github.com/srad/mediasink/internal/service/analysis"
)

type AnalysisHandler struct {
	analysis *analysissvc.Service
}

func NewAnalysisHandler(analysis *analysissvc.Service) *AnalysisHandler {
	return &AnalysisHandler{analysis: analysis}
}

func (h *AnalysisHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.analysis.GetResult(c.Request.Context(), db.RecordingID(id))
	if err != nil {
		writeError(c, err)
		return
	}

	response, err := responses.NewAnalysisResponse(uint(id), result)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
