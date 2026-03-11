package db

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
)

var (
	vecMu    sync.Mutex
	vecReady bool
)

// FrameVectorResult is returned by KNN searches on the frame_vectors table.
type FrameVectorResult struct {
	RowID       int64       `json:"rowId"`
	RecordingID RecordingID `json:"recordingId"`
	Timestamp   float64     `json:"timestamp"`
	Distance    float64     `json:"distance"`
}

// SimilarRecordingResult represents the best matching frame for a recording
// against a query vector.
type SimilarRecordingResult struct {
	RecordingID   RecordingID `json:"recordingId"`
	BestTimestamp float64     `json:"bestTimestamp"`
	Similarity    float64     `json:"similarity"`
}

// RecordingSimilarityEdge is a pairwise similarity between two recordings.
type RecordingSimilarityEdge struct {
	RecordingA RecordingID `json:"recordingA"`
	RecordingB RecordingID `json:"recordingB"`
	Similarity float64     `json:"similarity"`
}

// isNoSuchTable reports whether err is a "no such table" SQLite error.
// Used to treat a missing frame_vectors table as an empty result set rather
// than a hard failure — the table is created lazily on first analysis.
func isNoSuchTable(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "no such table")
}

// serializeFloat32 encodes a float32 slice to the raw IEEE 754 little-endian
// BLOB format that sqlite-vec expects.
func serializeFloat32(vec []float32) []byte {
	buf := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

// ensureVecTable creates the frame_vectors vec0 virtual table the first time
// it is needed. dim is the embedding dimension (number of float32 values).
// Idempotent: CREATE VIRTUAL TABLE IF NOT EXISTS is a no-op when the table
// already exists with the correct schema.
func ensureVecTable(dim int) error {
	vecMu.Lock()
	defer vecMu.Unlock()
	if vecReady {
		return nil
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec(fmt.Sprintf(`
		CREATE VIRTUAL TABLE IF NOT EXISTS frame_vectors USING vec0(
			embedding float[%d],
			recording_id    INTEGER,
			frame_index     INTEGER,
			frame_timestamp FLOAT
		)`, dim))
	if err == nil {
		vecReady = true
	}
	return err
}

// FrameVectorWriter streams frame vectors into sqlite-vec within a single
// transaction. Call Write for each frame in order, then Commit.
// On error, call Rollback. A nil writer (returned when not on SQLite) is safe
// to call on all methods — they all no-op.
type FrameVectorWriter struct {
	recordingID RecordingID
	tx          *sql.Tx
	stmt        *sql.Stmt
	idx         int
}

// NewFrameVectorWriter opens a transaction and prepares the insert statement.
// Returns nil, nil when the database is not SQLite (no-op path).
func NewFrameVectorWriter(recordingID RecordingID, dim int) (*FrameVectorWriter, error) {
	if err := ensureVecTable(dim); err != nil {
		return nil, fmt.Errorf("NewFrameVectorWriter: table init: %w", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		return nil, err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO frame_vectors(embedding, recording_id, frame_index, frame_timestamp)
		VALUES (?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	return &FrameVectorWriter{recordingID: recordingID, tx: tx, stmt: stmt}, nil
}

// Write inserts one frame vector. timestamp is the frame's position in seconds.
func (w *FrameVectorWriter) Write(vec []float32, timestamp float64) error {
	if w == nil {
		return nil
	}
	_, err := w.stmt.Exec(serializeFloat32(vec), uint(w.recordingID), w.idx, timestamp)
	if err != nil {
		return fmt.Errorf("FrameVectorWriter.Write index %d: %w", w.idx, err)
	}
	w.idx++
	return nil
}

// Commit finalises the transaction.
func (w *FrameVectorWriter) Commit() error {
	if w == nil {
		return nil
	}
	w.stmt.Close()
	return w.tx.Commit()
}

// Rollback aborts the transaction.
func (w *FrameVectorWriter) Rollback() {
	if w == nil {
		return
	}
	w.stmt.Close()
	w.tx.Rollback()
}

// DeleteFrameVectorsByRecordingID removes every stored frame vector for the
// given recording. Called before re-running analysis on a recording so stale
// vectors do not accumulate.
func DeleteFrameVectorsByRecordingID(recordingID RecordingID) error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	_, err = sqlDB.Exec(`DELETE FROM frame_vectors WHERE recording_id = ?`, uint(recordingID))
	if isNoSuchTable(err) {
		return nil
	}
	return err
}

// QueryConsecutiveSimilarities returns the cosine similarity between each
// consecutive pair of frames for the given recording, ordered by frame_index.
// Returns (timestamps, similarities) where timestamps[i] is the timestamp of
// the second frame in pair i (i.e. frameTimestamps[i+1] in the original list).
func QueryConsecutiveSimilarities(recordingID RecordingID) ([]float64, []float64, error) {
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, nil, err
	}
	rows, err := sqlDB.Query(`
		SELECT f2.frame_timestamp,
		       1.0 - vec_distance_cosine(f1.embedding, f2.embedding)
		FROM frame_vectors f1
		JOIN frame_vectors f2
		  ON f2.recording_id = f1.recording_id
		 AND f2.frame_index  = f1.frame_index + 1
		WHERE f1.recording_id = ?
		ORDER BY f1.frame_index
	`, uint(recordingID))
	if isNoSuchTable(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var timestamps, similarities []float64
	for rows.Next() {
		var ts, sim float64
		if err := rows.Scan(&ts, &sim); err != nil {
			return nil, nil, err
		}
		timestamps = append(timestamps, ts)
		similarities = append(similarities, sim)
	}
	return timestamps, similarities, rows.Err()
}

// SearchSimilarFrames performs a K-nearest-neighbour search over all stored
// frame vectors and returns the k closest frames by L2 distance.
func SearchSimilarFrames(queryVector []float32, k int) ([]FrameVectorResult, error) {
	if err := ensureVecTable(len(queryVector)); err != nil {
		return nil, fmt.Errorf("SearchSimilarFrames: table init: %w", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	rows, err := sqlDB.Query(`
		SELECT rowid, distance, recording_id, frame_timestamp
		FROM frame_vectors
		WHERE embedding MATCH ?
		  AND k = ?
		ORDER BY distance
	`, serializeFloat32(queryVector), k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanVectorResults(rows)
}

// SearchSimilarFramesByRecording performs KNN restricted to a single recording.
func SearchSimilarFramesByRecording(recordingID RecordingID, queryVector []float32, k int) ([]FrameVectorResult, error) {
	if err := ensureVecTable(len(queryVector)); err != nil {
		return nil, fmt.Errorf("SearchSimilarFramesByRecording: table init: %w", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	rows, err := sqlDB.Query(`
		SELECT rowid, distance, recording_id, frame_timestamp
		FROM frame_vectors
		WHERE embedding MATCH ?
		  AND k = ?
		  AND recording_id = ?
		ORDER BY distance
	`, serializeFloat32(queryVector), k, uint(recordingID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanVectorResults(rows)
}

func scanVectorResults(rows *sql.Rows) ([]FrameVectorResult, error) {
	var results []FrameVectorResult
	for rows.Next() {
		var r FrameVectorResult
		var recID uint
		if err := rows.Scan(&r.RowID, &r.Distance, &recID, &r.Timestamp); err != nil {
			return nil, err
		}
		r.RecordingID = RecordingID(recID)
		results = append(results, r)
	}
	return results, rows.Err()
}

// SearchSimilarRecordingsByVector returns one best-matching frame per recording,
// sorted by cosine similarity descending.
func SearchSimilarRecordingsByVector(queryVector []float32, minSimilarity float64, limit int) ([]SimilarRecordingResult, error) {
	if len(queryVector) == 0 {
		return nil, fmt.Errorf("query vector must not be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if err := ensureVecTable(len(queryVector)); err != nil {
		return nil, fmt.Errorf("SearchSimilarRecordingsByVector: table init: %w", err)
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}

	// Use sqlite-vec's optimized MATCH index with a safely bounded k=4000.
	// The maximum allowed limit for k in sqlite-vec is 4096.
	// This retrieves the 4000 closest frames instantly, which we then group
	// by recording_id to return the top distinct matching videos.
	rows, err := sqlDB.Query(`
		WITH knn AS (
		  SELECT recording_id,
		         frame_timestamp,
		         1.0 - distance AS similarity
		  FROM frame_vectors
		  WHERE embedding MATCH ? AND k = 4000
		),
		ranked AS (
		  SELECT recording_id,
		         frame_timestamp,
		         similarity,
		         ROW_NUMBER() OVER (
		           PARTITION BY recording_id
		           ORDER BY similarity DESC
		         ) AS rn
		  FROM knn
		  WHERE similarity >= ?
		)
		SELECT recording_id, frame_timestamp, similarity
		FROM ranked
		WHERE rn = 1
		ORDER BY similarity DESC
		LIMIT ?
	`, serializeFloat32(queryVector), minSimilarity, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SimilarRecordingResult
	for rows.Next() {
		var recID uint
		var r SimilarRecordingResult
		if err := rows.Scan(&recID, &r.BestTimestamp, &r.Similarity); err != nil {
			return nil, err
		}
		r.RecordingID = RecordingID(recID)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListRecordingIDsWithFrameVectors returns distinct recording IDs that have
// stored frame vectors.
func ListRecordingIDsWithFrameVectors(limit int) ([]RecordingID, error) {
	if limit <= 0 {
		limit = 500
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}
	rows, err := sqlDB.Query(`
		SELECT DISTINCT recording_id
		FROM frame_vectors
		ORDER BY recording_id
		LIMIT ?
	`, limit)
	if isNoSuchTable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RecordingID
	for rows.Next() {
		var recID uint
		if err := rows.Scan(&recID); err != nil {
			return nil, err
		}
		out = append(out, RecordingID(recID))
	}
	return out, rows.Err()
}

// QueryRecordingSimilarityEdges computes pairwise recording similarities by
// fetching a small sample of frame vectors for each recording into Go memory,
// and manually calculating the distance array to avoid O(N^2) SQLite joins.
func QueryRecordingSimilarityEdges(minSimilarity float64, recordingIDs []RecordingID, limit int) ([]RecordingSimilarityEdge, error) {
	if limit <= 0 {
		limit = 20000
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}

	// 1. If recordingIDs is empty, fetch all distinct recording IDs.
	if len(recordingIDs) == 0 {
		ids, err := ListRecordingIDsWithFrameVectors(0)
		if err != nil {
			return nil, err
		}
		recordingIDs = ids
	}

	if len(recordingIDs) < 2 {
		return nil, nil // Nothing to group
	}

	// Make a set of valid recording IDs for fast filtering
	validRecs := make(map[RecordingID]bool, len(recordingIDs))
	for _, id := range recordingIDs {
		validRecs[id] = true
	}

	// 2. Fetch up to 5 evenly spaced sample vectors for each requested recording
	samples := make(map[RecordingID][][]float32)
	for _, recID := range recordingIDs {
		var maxIndex sql.NullInt32
		err := sqlDB.QueryRow("SELECT MAX(frame_index) FROM frame_vectors WHERE recording_id = ?", uint(recID)).Scan(&maxIndex)
		if err != nil || !maxIndex.Valid {
			if isNoSuchTable(err) {
				return nil, nil
			}
			continue
		}

		step := int(maxIndex.Int32) / 5
		if step == 0 {
			step = 1
		}

		rows, err := sqlDB.Query(`
			SELECT embedding FROM frame_vectors 
			WHERE recording_id = ? AND frame_index IN (?, ?, ?, ?, ?)
		`, uint(recID), 0, step, step*2, step*3, step*4)

		if err != nil {
			continue
		}

		for rows.Next() {
			var b []byte
			if err := rows.Scan(&b); err == nil {
				// Convert raw IEEE-754 bytes to float32
				floats := make([]float32, len(b)/4)
				for i := range floats {
					floats[i] = math.Float32frombits(uint32(b[i*4]) | uint32(b[i*4+1])<<8 | uint32(b[i*4+2])<<16 | uint32(b[i*4+3])<<24)
				}
				samples[recID] = append(samples[recID], floats)
			}
		}
		rows.Close()
	}

	// 3. Compute cosine similarity in memory
	var out []RecordingSimilarityEdge

	// Create predictable slice for pair computation (i < j)
	var validIDs []RecordingID
	for id := range samples {
		validIDs = append(validIDs, id)
	}

	for i := 0; i < len(validIDs); i++ {
		recA := validIDs[i]
		sA := samples[recA]

		for j := i + 1; j < len(validIDs); j++ {
			recB := validIDs[j]
			sB := samples[recB]

			maxSim := -1.0
			for _, a := range sA {
				for _, b := range sB {
					// Cosine similarity
					var dot float64
					var normA float64
					var normB float64

					for k := 0; k < len(a) && k < len(b); k++ {
						vA, vB := float64(a[k]), float64(b[k])
						dot += vA * vB
						normA += vA * vA
						normB += vB * vB
					}

					var sim float64
					if normA > 0 && normB > 0 {
						sim = dot / (math.Sqrt(normA) * math.Sqrt(normB))
					}

					if sim > maxSim {
						maxSim = sim
					}
				}
			}

			if maxSim >= minSimilarity {
				var rA, rB RecordingID
				if recA < recB {
					rA, rB = recA, recB
				} else {
					rA, rB = recB, recA
				}
				out = append(out, RecordingSimilarityEdge{
					RecordingA: rA,
					RecordingB: rB,
					Similarity: maxSim,
				})
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(out, func(i, j int) bool {
		return out[i].Similarity > out[j].Similarity
	})

	if len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}
