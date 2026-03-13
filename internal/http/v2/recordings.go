package v2

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/db"
	analysissvc "github.com/srad/mediasink/internal/service/analysis"
	recordingssvc "github.com/srad/mediasink/internal/service/recordings"
)

type RecordingsHandler struct {
	recordings *recordingssvc.Service
	analysis   *analysissvc.Service
}

func NewRecordingsHandler(recordings *recordingssvc.Service, analysis *analysissvc.Service) *RecordingsHandler {
	return &RecordingsHandler{
		recordings: recordings,
		analysis:   analysis,
	}
}

func (h *RecordingsHandler) List(c *gin.Context) {
	recordings, err := h.recordings.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, recordings)
}

func (h *RecordingsHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	recording, err := h.recordings.Get(c.Request.Context(), db.RecordingID(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, recording)
}

func (h *RecordingsHandler) CreatePreviewJob(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.recordings.CreatePreviewJob(c.Request.Context(), db.RecordingID(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, job)
}

func (h *RecordingsHandler) CreateAnalysisJob(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, previewState, err := h.analysis.CreateJob(c.Request.Context(), db.RecordingID(id))
	if err != nil {
		writeError(c, err)
		return
	}
	if previewState.NeedsRegeneration {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{
			"error":   "preview must exist before analysis can be queued",
			"preview": previewState,
		})
		return
	}
	c.JSON(http.StatusAccepted, job)
}
