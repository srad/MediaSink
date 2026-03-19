package segments

import (
	"fmt"
	"math"
	"sort"

	"github.com/srad/mediasink/internal/analysis/smoothing"
	"github.com/srad/mediasink/internal/db"
)

const chapterKind = "chapter"

type EmbeddingFrame struct {
	Timestamp float64
	Values    []float32
}

type ChapterConfig struct {
	WindowDuration     float64
	SmoothingWindow    int
	PeakMADMultiplier  float64
	MinPeakScore       float64
	MinChapterDuration float64
	MaxChapterDuration float64
}

func DefaultChapterConfig() ChapterConfig {
	return ChapterConfig{
		WindowDuration:     45,
		SmoothingWindow:    5,
		PeakMADMultiplier:  4,
		MinPeakScore:       0.12,
		MinChapterDuration: 120,
		MaxChapterDuration: 20 * 60,
	}
}

type boundaryCandidate struct {
	SplitIndex int
	Timestamp  float64
	Score      float64
	Confidence float64
	Peak       bool
}

type segmentRange struct {
	StartIndex int
	EndIndex   int
	StartTime  float64
	EndTime    float64
}

func (s segmentRange) Duration() float64 {
	return s.EndTime - s.StartTime
}

func DetectChapters(frames []EmbeddingFrame, totalDuration float64, config ChapterConfig) ([]db.SegmentInfo, error) {
	if len(frames) == 0 {
		return []db.SegmentInfo{}, nil
	}

	effectiveEnd := totalDuration
	if effectiveEnd < frames[len(frames)-1].Timestamp {
		effectiveEnd = frames[len(frames)-1].Timestamp
	}

	if len(frames) == 1 {
		return []db.SegmentInfo{{
			Kind:                    chapterKind,
			StartTime:               0,
			EndTime:                 effectiveEnd,
			Confidence:              0,
			RepresentativeTimestamp: frames[0].Timestamp,
		}}, nil
	}

	candidates, threshold, err := detectBoundaryCandidates(frames, config)
	if err != nil {
		return nil, err
	}

	splits := selectPrimarySplits(candidates, effectiveThreshold(threshold, config))
	splits = mergeShortChapterSegments(frames, effectiveEnd, splits, config)
	splits = splitLongChapterSegments(frames, effectiveEnd, splits, candidates, config)

	return buildSegments(frames, effectiveEnd, splits, candidates), nil
}

func detectBoundaryCandidates(frames []EmbeddingFrame, config ChapterConfig) ([]boundaryCandidate, float64, error) {
	if len(frames) < 2 {
		return nil, 0, nil
	}
	if len(frames[0].Values) == 0 {
		return nil, 0, fmt.Errorf("empty embedding vectors")
	}

	dim := len(frames[0].Values)
	leftSum := make([]float64, dim)
	rightSum := make([]float64, dim)

	leftStart := 0
	leftCount := 0
	rightStart := 1
	rightEnd := 1
	rightCount := 0

	candidates := make([]boundaryCandidate, 0, len(frames)-1)
	for i := 1; i < len(frames); i++ {
		if len(frames[i].Values) != dim || len(frames[i-1].Values) != dim {
			return nil, 0, fmt.Errorf("inconsistent embedding dimension")
		}

		timestamp := frames[i].Timestamp

		addEmbedding(leftSum, frames[i-1].Values)
		leftCount++

		for leftStart < i && frames[leftStart].Timestamp < timestamp-config.WindowDuration {
			subEmbedding(leftSum, frames[leftStart].Values)
			leftStart++
			leftCount--
		}

		for rightStart < i {
			subEmbedding(rightSum, frames[rightStart].Values)
			rightStart++
			rightCount--
		}
		for rightEnd < len(frames) && frames[rightEnd].Timestamp < timestamp+config.WindowDuration {
			if len(frames[rightEnd].Values) != dim {
				return nil, 0, fmt.Errorf("inconsistent embedding dimension")
			}
			addEmbedding(rightSum, frames[rightEnd].Values)
			rightEnd++
			rightCount++
		}

		if leftCount == 0 || rightCount == 0 {
			continue
		}

		score := 1 - cosineSimilarity(leftSum, rightSum)
		if score < 0 {
			score = 0
		}
		candidates = append(candidates, boundaryCandidate{
			SplitIndex: i,
			Timestamp:  timestamp,
			Score:      score,
		})
	}

	if len(candidates) == 0 {
		return nil, 0, nil
	}

	scores := make([]float64, len(candidates))
	for i, candidate := range candidates {
		scores[i] = candidate.Score
	}

	smoother, err := smoothing.NewSmoothingMethod("gaussian")
	if err != nil {
		return nil, 0, err
	}
	smoothed := smoother.Smooth(scores, config.SmoothingWindow)
	threshold := effectiveThreshold(robustPeakThreshold(smoothed, config.PeakMADMultiplier), config)
	maxScore := maxValue(smoothed)

	for i := range candidates {
		candidates[i].Score = smoothed[i]
		candidates[i].Peak = isLocalMaximum(smoothed, i)
		candidates[i].Confidence = normalizeConfidence(smoothed[i], threshold, maxScore)
	}

	return candidates, threshold, nil
}

func selectPrimarySplits(candidates []boundaryCandidate, threshold float64) []int {
	splits := make([]int, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Peak || candidate.Confidence <= 0 || candidate.Score <= threshold {
			continue
		}
		splits = append(splits, candidate.SplitIndex)
	}
	return normalizeSplits(splits)
}

func mergeShortChapterSegments(frames []EmbeddingFrame, totalDuration float64, splits []int, config ChapterConfig) []int {
	for {
		ranges := buildSegmentRanges(frames, totalDuration, splits)
		if len(ranges) <= 1 {
			return splits
		}

		shortIndex := -1
		for i, segment := range ranges {
			if segment.Duration() < config.MinChapterDuration {
				shortIndex = i
				break
			}
		}
		if shortIndex == -1 {
			return splits
		}

		switch {
		case shortIndex == 0:
			splits = removeSplitAt(splits, 0)
		case shortIndex == len(ranges)-1:
			splits = removeSplitAt(splits, len(splits)-1)
		default:
			current := ranges[shortIndex]
			prev := ranges[shortIndex-1]
			next := ranges[shortIndex+1]

			prevSimilarity := centroidSimilarity(frames, current.StartIndex, current.EndIndex, prev.StartIndex, prev.EndIndex)
			nextSimilarity := centroidSimilarity(frames, current.StartIndex, current.EndIndex, next.StartIndex, next.EndIndex)
			if prevSimilarity >= nextSimilarity {
				splits = removeSplitValue(splits, current.StartIndex)
			} else {
				splits = removeSplitValue(splits, current.EndIndex)
			}
		}
	}
}

func splitLongChapterSegments(frames []EmbeddingFrame, totalDuration float64, splits []int, candidates []boundaryCandidate, config ChapterConfig) []int {
	used := make(map[int]struct{}, len(splits))
	for _, split := range splits {
		used[split] = struct{}{}
	}

	for {
		ranges := buildSegmentRanges(frames, totalDuration, splits)
		changed := false

		for _, segment := range ranges {
			if segment.Duration() <= config.MaxChapterDuration {
				continue
			}

			candidate, ok := strongestRemainingPeak(segment, candidates, used, config)
			if !ok {
				continue
			}

			splits = append(splits, candidate.SplitIndex)
			splits = normalizeSplits(splits)
			used[candidate.SplitIndex] = struct{}{}
			changed = true
			break
		}

		if !changed {
			return splits
		}
	}
}

func buildSegments(frames []EmbeddingFrame, totalDuration float64, splits []int, candidates []boundaryCandidate) []db.SegmentInfo {
	ranges := buildSegmentRanges(frames, totalDuration, splits)
	if len(ranges) == 0 {
		return []db.SegmentInfo{}
	}

	confBySplit := make(map[int]float64, len(candidates))
	for _, candidate := range candidates {
		confBySplit[candidate.SplitIndex] = candidate.Confidence
	}

	segments := make([]db.SegmentInfo, 0, len(ranges))
	for i, segment := range ranges {
		confidence := 0.0
		if i > 0 {
			confidence = math.Max(confidence, confBySplit[splits[i-1]])
		}
		if i < len(splits) {
			confidence = math.Max(confidence, confBySplit[splits[i]])
		}

		segments = append(segments, db.SegmentInfo{
			Kind:                    chapterKind,
			StartTime:               segment.StartTime,
			EndTime:                 segment.EndTime,
			Confidence:              clamp(confidence, 0, 1),
			RepresentativeTimestamp: representativeTimestamp(frames, segment.StartIndex, segment.EndIndex, segment.StartTime, segment.EndTime),
		})
	}

	return segments
}

func buildSegmentRanges(frames []EmbeddingFrame, totalDuration float64, splits []int) []segmentRange {
	normalized := normalizeSplits(splits)
	ranges := make([]segmentRange, 0, len(normalized)+1)

	startIndex := 0
	startTime := 0.0
	for _, split := range normalized {
		if split <= startIndex || split >= len(frames) {
			continue
		}
		endTime := frames[split].Timestamp
		ranges = append(ranges, segmentRange{
			StartIndex: startIndex,
			EndIndex:   split,
			StartTime:  startTime,
			EndTime:    endTime,
		})
		startIndex = split
		startTime = frames[split].Timestamp
	}

	ranges = append(ranges, segmentRange{
		StartIndex: startIndex,
		EndIndex:   len(frames),
		StartTime:  startTime,
		EndTime:    totalDuration,
	})
	return ranges
}

func strongestRemainingPeak(segment segmentRange, candidates []boundaryCandidate, used map[int]struct{}, config ChapterConfig) (boundaryCandidate, bool) {
	best := boundaryCandidate{}
	found := false

	for _, candidate := range candidates {
		if !candidate.Peak {
			continue
		}
		if candidate.Confidence <= 0 || candidate.Score < config.MinPeakScore {
			continue
		}
		if _, exists := used[candidate.SplitIndex]; exists {
			continue
		}
		if candidate.SplitIndex <= segment.StartIndex || candidate.SplitIndex >= segment.EndIndex {
			continue
		}
		if candidate.Timestamp-segment.StartTime < config.MinChapterDuration || segment.EndTime-candidate.Timestamp < config.MinChapterDuration {
			continue
		}
		if !found || candidate.Score > best.Score {
			best = candidate
			found = true
		}
	}

	return best, found
}

func centroidSimilarity(frames []EmbeddingFrame, aStart, aEnd, bStart, bEnd int) float64 {
	a := centroid(frames, aStart, aEnd)
	b := centroid(frames, bStart, bEnd)
	return cosineSimilarity(a, b)
}

func centroid(frames []EmbeddingFrame, start, end int) []float64 {
	if start >= end || start < 0 || end > len(frames) {
		return nil
	}
	sum := make([]float64, len(frames[start].Values))
	for i := start; i < end; i++ {
		addEmbedding(sum, frames[i].Values)
	}
	return sum
}

func representativeTimestamp(frames []EmbeddingFrame, start, end int, startTime, endTime float64) float64 {
	if start >= end {
		return startTime
	}
	target := startTime + (endTime-startTime)/2
	best := frames[start].Timestamp
	bestDistance := math.Abs(best - target)
	for i := start + 1; i < end; i++ {
		distance := math.Abs(frames[i].Timestamp - target)
		if distance < bestDistance {
			best = frames[i].Timestamp
			bestDistance = distance
		}
	}
	return best
}

func robustPeakThreshold(scores []float64, multiplier float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	if multiplier <= 0 {
		multiplier = 3
	}

	medianScore := median(scores)
	absDeviations := make([]float64, len(scores))
	for i, score := range scores {
		absDeviations[i] = math.Abs(score - medianScore)
	}
	return medianScore + multiplier*median(absDeviations)
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

func isLocalMaximum(scores []float64, idx int) bool {
	if idx < 0 || idx >= len(scores) {
		return false
	}
	leftOK := idx == 0 || scores[idx] >= scores[idx-1]
	rightOK := idx == len(scores)-1 || scores[idx] > scores[idx+1]
	return leftOK && rightOK
}

func normalizeConfidence(score, threshold, maxScore float64) float64 {
	if score <= threshold || maxScore <= threshold {
		return 0
	}
	return clamp((score-threshold)/(maxScore-threshold), 0, 1)
}

func effectiveThreshold(threshold float64, config ChapterConfig) float64 {
	return math.Max(threshold, config.MinPeakScore)
}

func normalizeSplits(splits []int) []int {
	if len(splits) == 0 {
		return nil
	}
	copySplits := append([]int(nil), splits...)
	sort.Ints(copySplits)

	out := copySplits[:0]
	prev := -1
	for _, split := range copySplits {
		if split == prev {
			continue
		}
		out = append(out, split)
		prev = split
	}
	return out
}

func removeSplitAt(splits []int, idx int) []int {
	if idx < 0 || idx >= len(splits) {
		return splits
	}
	out := append([]int(nil), splits[:idx]...)
	out = append(out, splits[idx+1:]...)
	return out
}

func removeSplitValue(splits []int, split int) []int {
	for i, current := range splits {
		if current == split {
			return removeSplitAt(splits, i)
		}
	}
	return splits
}

func addEmbedding(sum []float64, values []float32) {
	for i, value := range values {
		sum[i] += float64(value)
	}
}

func subEmbedding(sum []float64, values []float32) {
	for i, value := range values {
		sum[i] -= float64(value)
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}

	dot := 0.0
	normA := 0.0
	normB := 0.0
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func maxValue(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maximum := values[0]
	for _, value := range values[1:] {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func clamp(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
