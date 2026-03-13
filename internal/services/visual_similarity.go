package services

import (
	"context"
	"fmt"
	"image"
	"io"
	"sort"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/srad/mediasink/internal/analysis/detectors"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/store/vector"
)

// SimilarRecordingMatch is an aggregated recording-level similarity result.
type SimilarRecordingMatch struct {
	RecordingID   db.RecordingID
	Similarity    float64
	BestTimestamp float64
}

// SimilarRecordingGroup is a cluster of recordings connected by similarity edges.
type SimilarRecordingGroup struct {
	RecordingIDs  []db.RecordingID
	MaxSimilarity float64
}

// NormalizeSimilarityThreshold supports both 0..1 and 0..100 slider values.
func NormalizeSimilarityThreshold(v float64) (float64, error) {
	if v > 1.0 && v <= 100.0 {
		v = v / 100.0
	}
	if v < 0 || v > 1 {
		return 0, fmt.Errorf("similarity must be in range 0..1 or 0..100")
	}
	return v, nil
}

// SearchSimilarRecordingsByImage finds recordings visually similar to an uploaded image.
func SearchSimilarRecordingsByImage(r io.Reader, minSimilarity float64, limit int) ([]SimilarRecordingMatch, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	sceneDetector, err := detectors.CreateSceneDetector(detectors.DetectorTypeOnnxMobileNetV3Large)
	if err != nil {
		return nil, fmt.Errorf("failed to create feature extractor: %w", err)
	}
	featureExtractor, ok := sceneDetector.(FeatureExtractor)
	if !ok {
		return nil, fmt.Errorf("scene detector does not expose ExtractFeatures")
	}

	vec, err := featureExtractor.ExtractFeatures(img)
	if err != nil {
		return nil, fmt.Errorf("failed to extract query embedding: %w", err)
	}

	results, err := vector.Default().SearchSimilarRecordings(context.Background(), vec, minSimilarity, limit)
	if err != nil {
		return nil, err
	}

	out := make([]SimilarRecordingMatch, 0, len(results))
	for _, r := range results {
		out = append(out, SimilarRecordingMatch{
			RecordingID:   r.RecordingID,
			Similarity:    r.Similarity,
			BestTimestamp: r.BestTimestamp,
		})
	}
	return out, nil
}

// GroupSimilarRecordings builds similarity-based recording clusters.
func GroupSimilarRecordings(minSimilarity float64, recordingIDs []db.RecordingID, pairLimit int, includeSingletons bool) ([]SimilarRecordingGroup, error) {
	ids := recordingIDs
	if len(ids) == 0 {
		autoIDs, err := vector.Default().ListRecordingIDs(context.Background(), 1000)
		if err != nil {
			return nil, err
		}
		ids = autoIDs
	}
	if len(ids) == 0 {
		return []SimilarRecordingGroup{}, nil
	}

	edges, err := vector.Default().QueryRecordingSimilarityEdges(context.Background(), minSimilarity, ids, pairLimit)
	if err != nil {
		return nil, err
	}

	type dsu struct {
		parent map[db.RecordingID]db.RecordingID
	}
	var find func(d *dsu, x db.RecordingID) db.RecordingID
	find = func(d *dsu, x db.RecordingID) db.RecordingID {
		p := d.parent[x]
		if p == x {
			return x
		}
		root := find(d, p)
		d.parent[x] = root
		return root
	}
	union := func(d *dsu, a, b db.RecordingID) {
		ra := find(d, a)
		rb := find(d, b)
		if ra != rb {
			d.parent[rb] = ra
		}
	}

	d := &dsu{parent: make(map[db.RecordingID]db.RecordingID, len(ids))}
	for _, id := range ids {
		d.parent[id] = id
	}
	for _, e := range edges {
		union(d, e.RecordingA, e.RecordingB)
	}

	componentIDs := make(map[db.RecordingID][]db.RecordingID)
	componentMax := make(map[db.RecordingID]float64)
	for _, id := range ids {
		root := find(d, id)
		componentIDs[root] = append(componentIDs[root], id)
	}
	for _, e := range edges {
		root := find(d, e.RecordingA)
		if e.Similarity > componentMax[root] {
			componentMax[root] = e.Similarity
		}
	}

	var groups []SimilarRecordingGroup
	for root, members := range componentIDs {
		if !includeSingletons && len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool { return members[i] < members[j] })
		groups = append(groups, SimilarRecordingGroup{
			RecordingIDs:  members,
			MaxSimilarity: componentMax[root],
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		if len(groups[i].RecordingIDs) == len(groups[j].RecordingIDs) {
			return groups[i].MaxSimilarity > groups[j].MaxSimilarity
		}
		return len(groups[i].RecordingIDs) > len(groups[j].RecordingIDs)
	})
	return groups, nil
}
