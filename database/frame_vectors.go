package database

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"os"
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

// isSQLite reports whether the configured database adapter is SQLite.
// The frame_vectors vec0 virtual table is SQLite-only.
func isSQLite() bool {
	a := os.Getenv("DB_ADAPTER")
	return a == "" || a == "sqlite" || a == "sqlite3"
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
	if !isSQLite() {
		return nil, nil
	}
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
	if !isSQLite() {
		return nil
	}
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
	if !isSQLite() {
		return nil, nil, fmt.Errorf("QueryConsecutiveSimilarities requires SQLite")
	}
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
	if !isSQLite() {
		return nil, fmt.Errorf("SearchSimilarFrames requires SQLite")
	}
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
	if !isSQLite() {
		return nil, fmt.Errorf("SearchSimilarFramesByRecording requires SQLite")
	}
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
	if !isSQLite() {
		return nil, fmt.Errorf("SearchSimilarRecordingsByVector requires SQLite")
	}
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

	rows, err := sqlDB.Query(`
		WITH scored AS (
		  SELECT recording_id,
		         frame_timestamp,
		         1.0 - vec_distance_cosine(embedding, ?) AS similarity
		  FROM frame_vectors
		),
		ranked AS (
		  SELECT recording_id,
		         frame_timestamp,
		         similarity,
		         ROW_NUMBER() OVER (
		           PARTITION BY recording_id
		           ORDER BY similarity DESC
		         ) AS rn
		  FROM scored
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
	if !isSQLite() {
		return nil, fmt.Errorf("ListRecordingIDsWithFrameVectors requires SQLite")
	}
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
// taking the best frame-to-frame cosine similarity between each recording pair.
// If recordingIDs is non-empty, only those recordings are considered.
func QueryRecordingSimilarityEdges(minSimilarity float64, recordingIDs []RecordingID, limit int) ([]RecordingSimilarityEdge, error) {
	if !isSQLite() {
		return nil, fmt.Errorf("QueryRecordingSimilarityEdges requires SQLite")
	}
	if limit <= 0 {
		limit = 20000
	}
	sqlDB, err := DB.DB()
	if err != nil {
		return nil, err
	}

	var b strings.Builder
	args := make([]interface{}, 0, len(recordingIDs)*2+2)

	b.WriteString(`
		SELECT f1.recording_id AS recording_a,
		       f2.recording_id AS recording_b,
		       MAX(1.0 - vec_distance_cosine(f1.embedding, f2.embedding)) AS similarity
		FROM frame_vectors f1
		JOIN frame_vectors f2
		  ON f1.recording_id < f2.recording_id
	`)

	if len(recordingIDs) > 0 {
		ids := make([]interface{}, 0, len(recordingIDs))
		ph := make([]string, 0, len(recordingIDs))
		for _, id := range recordingIDs {
			ph = append(ph, "?")
			ids = append(ids, uint(id))
		}
		inList := strings.Join(ph, ",")
		b.WriteString(` WHERE f1.recording_id IN (` + inList + `) AND f2.recording_id IN (` + inList + `)`)
		args = append(args, ids...)
		args = append(args, ids...)
	}

	b.WriteString(`
		GROUP BY f1.recording_id, f2.recording_id
		HAVING similarity >= ?
		ORDER BY similarity DESC
		LIMIT ?
	`)
	args = append(args, minSimilarity, limit)

	rows, err := sqlDB.Query(b.String(), args...)
	if isNoSuchTable(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []RecordingSimilarityEdge
	for rows.Next() {
		var a, c uint
		var sim float64
		if err := rows.Scan(&a, &c, &sim); err != nil {
			return nil, err
		}
		out = append(out, RecordingSimilarityEdge{
			RecordingA: RecordingID(a),
			RecordingB: RecordingID(c),
			Similarity: sim,
		})
	}
	return out, rows.Err()
}
