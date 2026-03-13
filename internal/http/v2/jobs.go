package v2

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/models/responses"
	jobsvc "github.com/srad/mediasink/internal/service/jobs"
)

type JobsHandler struct {
	jobs *jobsvc.Service
}

func NewJobsHandler(jobs *jobsvc.Service) *JobsHandler {
	return &JobsHandler{jobs: jobs}
}

func (h *JobsHandler) List(c *gin.Context) {
	skip, err := parseIntQuery(c, "skip", 0)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	take, err := parseIntQuery(c, "take", 50)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statuses := parseJobStatuses(splitQueryValues(c.QueryArray("status")))
	order := db.JobOrder(strings.ToUpper(c.DefaultQuery("order", string(db.JobOrderDESC))))

	jobs, totalCount, err := h.jobs.List(c.Request.Context(), skip, take, statuses, order)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, responses.JobsResponse{
		Jobs:       jobs,
		TotalCount: totalCount,
		Skip:       skip,
		Take:       take,
	})
}

func (h *JobsHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.jobs.Get(c.Request.Context(), uint(id))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, job)
}

func parseJobStatuses(values []string) []db.JobStatus {
	if len(values) == 0 {
		return []db.JobStatus{
			db.StatusJobOpen,
			db.StatusJobCompleted,
			db.StatusJobError,
			db.StatusJobCanceled,
		}
	}

	statuses := make([]db.JobStatus, 0, len(values))
	for _, value := range values {
		statuses = append(statuses, db.JobStatus(value))
	}
	return statuses
}
