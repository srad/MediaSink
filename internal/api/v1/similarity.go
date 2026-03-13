package v1

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/srad/mediasink/internal/app"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/models/requests"
	"github.com/srad/mediasink/internal/models/responses"
	"github.com/srad/mediasink/internal/services"
	"github.com/srad/mediasink/internal/store/vector"
)

// SearchSimilarVideosByImage godoc
// @Summary     Search visually similar videos by image
// @Description Upload a picture and return visually similar videos using frame embeddings.
// @Tags        analysis
// @Accept      multipart/form-data
// @Produce     json
// @Param       file formData file true "Query image file"
// @Param       similarity formData number false "Similarity threshold (0..1 or 0..100), default 0.8"
// @Param       limit formData int false "Max results (1..200), default 50"
// @Success     200 {object} responses.VisualSearchResponse
// @Failure     400 {} string "Invalid input"
// @Failure     500 {} string "Error message"
// @Router      /analysis/search/image [post]
func SearchSimilarVideosByImage(c *gin.Context) {
	appG := app.Gin{C: c}

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}
	defer file.Close()

	similarity := 0.8
	if raw := c.PostForm("similarity"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			appG.Error(http.StatusBadRequest, err)
			return
		}
		v, err = services.NormalizeSimilarityThreshold(v)
		if err != nil {
			appG.Error(http.StatusBadRequest, err)
			return
		}
		similarity = v
	}

	limit := 50
	if raw := c.PostForm("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			appG.Error(http.StatusBadRequest, err)
			return
		}
		if v < 1 || v > 200 {
			appG.Error(http.StatusBadRequest, fmt.Errorf("limit must be in range 1..200"))
			return
		}
		limit = v
	}

	matches, err := services.SearchSimilarRecordingsByImage(file, similarity, limit)
	if err != nil {
		log.Warnf("[SearchSimilarVideosByImage] %v", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
		return
	}

	ids := make([]db.RecordingID, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.RecordingID)
	}
	recordings, err := db.FindRecordingsByIDs(ids)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}
	recByID := make(map[db.RecordingID]*db.Recording, len(recordings))
	for _, rec := range recordings {
		recByID[rec.RecordingID] = rec
	}

	results := make([]responses.SimilarVideoMatch, 0, len(matches))
	for _, m := range matches {
		rec := recByID[m.RecordingID]
		if rec == nil {
			continue
		}
		results = append(results, responses.SimilarVideoMatch{
			Recording:     rec,
			Similarity:    m.Similarity,
			BestTimestamp: m.BestTimestamp,
		})
	}

	appG.Response(http.StatusOK, responses.VisualSearchResponse{
		SimilarityThreshold: similarity,
		Limit:               limit,
		Results:             results,
	})
}

// GroupSimilarVideos godoc
// @Summary     Group visually similar videos
// @Description Build similarity clusters from analyzed frame vectors.
// @Tags        analysis
// @Accept      json
// @Produce     json
// @Param       request body requests.SimilarityGroupRequest true "Grouping request"
// @Success     200 {object} responses.SimilarityGroupsResponse
// @Failure     400 {} string "Invalid input"
// @Failure     500 {} string "Error message"
// @Router      /analysis/group [post]
func GroupSimilarVideos(c *gin.Context) {
	appG := app.Gin{C: c}

	var req requests.SimilarityGroupRequest
	if err := c.BindJSON(&req); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}

	similarity := 0.8
	if req.Similarity != nil {
		v, err := services.NormalizeSimilarityThreshold(*req.Similarity)
		if err != nil {
			appG.Error(http.StatusBadRequest, err)
			return
		}
		similarity = v
	}

	pairLimit := req.PairLimit
	if pairLimit == 0 {
		pairLimit = 20000
	}
	if pairLimit < 1 || pairLimit > 100000 {
		appG.Error(http.StatusBadRequest, fmt.Errorf("pairLimit must be in range 1..100000"))
		return
	}

	ids := make([]db.RecordingID, 0, len(req.RecordingIDs))
	for _, id := range req.RecordingIDs {
		if id == 0 {
			continue
		}
		ids = append(ids, db.RecordingID(id))
	}

	groups, err := services.GroupSimilarRecordings(similarity, ids, pairLimit, req.IncludeSingletons)
	if err != nil {
		log.Warnf("[GroupSimilarVideos] %v", err)
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, err.Error())
		return
	}

	// Load recordings once and map by ID for assembling groups.
	idSet := make(map[db.RecordingID]struct{})
	for _, g := range groups {
		for _, id := range g.RecordingIDs {
			idSet[id] = struct{}{}
		}
	}
	allIDs := make([]db.RecordingID, 0, len(idSet))
	for id := range idSet {
		allIDs = append(allIDs, id)
	}
	recs, err := db.FindRecordingsByIDs(allIDs)
	if err != nil {
		appG.Error(http.StatusInternalServerError, err)
		return
	}
	recByID := make(map[db.RecordingID]*db.Recording, len(recs))
	for _, rec := range recs {
		recByID[rec.RecordingID] = rec
	}

	outGroups := make([]responses.SimilarVideoGroup, 0, len(groups))
	for i, g := range groups {
		videos := make([]*db.Recording, 0, len(g.RecordingIDs))
		for _, id := range g.RecordingIDs {
			if rec := recByID[id]; rec != nil {
				videos = append(videos, rec)
			}
		}
		outGroups = append(outGroups, responses.SimilarVideoGroup{
			GroupID:       i + 1,
			MaxSimilarity: g.MaxSimilarity,
			Videos:        videos,
		})
	}

	analyzedIDs, _ := vector.Default().ListRecordingIDs(context.Background(), 1000)

	appG.Response(http.StatusOK, responses.SimilarityGroupsResponse{
		SimilarityThreshold: similarity,
		GroupCount:          len(outGroups),
		Groups:              outGroups,
		AnalyzedCount:       len(analyzedIDs),
	})
}
