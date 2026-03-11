package db

import (
	"math"
	"os"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlite-vec requires embedding dimension >= 8 (chunk_size = dim/8 must be > 0).
const testDim = 16

// setupVecDB initialises an in-memory SQLite database with the sqlite-vec
// extension loaded and assigns it to the package-level DB variable.
// It also resets the vecReady flag so that ensureVecTable runs fresh.
func setupVecDB(t *testing.T) {
	t.Helper()
	sqlite_vec.Auto()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	DB = db

	// Reset the lazy-init flag so each test gets a fresh table.
	vecMu.Lock()
	vecReady = false
	vecMu.Unlock()
}

// unitVec returns a testDim-element vector with 1 at position pos and 0 elsewhere.
func unitVec(pos int) []float32 {
	v := make([]float32, testDim)
	v[pos] = 1
	return v
}

func TestFrameVectorWriter_CommitAndQuery(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recID = RecordingID(1)
	// Two identical unit vectors then one orthogonal to them.
	vecs := [][]float32{unitVec(0), unitVec(0), unitVec(1)}
	timestamps := []float64{0.0, 1.0, 2.0}

	w, err := NewFrameVectorWriter(recID, testDim)
	if err != nil {
		t.Fatalf("NewFrameVectorWriter: %v", err)
	}
	for i, v := range vecs {
		if err := w.Write(v, timestamps[i]); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	ts, sims, err := QueryConsecutiveSimilarities(recID)
	if err != nil {
		t.Fatalf("QueryConsecutiveSimilarities: %v", err)
	}

	if len(sims) != 2 {
		t.Fatalf("expected 2 similarities, got %d", len(sims))
	}
	if len(ts) != 2 {
		t.Fatalf("expected 2 timestamps, got %d", len(ts))
	}

	// vecs[0] vs vecs[1]: identical unit vectors → similarity ≈ 1.0
	if math.Abs(sims[0]-1.0) > 0.001 {
		t.Errorf("pair 0-1: expected similarity ~1.0, got %.6f", sims[0])
	}
	if ts[0] != 1.0 {
		t.Errorf("pair 0-1 timestamp: expected 1.0, got %.1f", ts[0])
	}

	// vecs[1] vs vecs[2]: orthogonal unit vectors → similarity ≈ 0.0
	if math.Abs(sims[1]) > 0.001 {
		t.Errorf("pair 1-2: expected similarity ~0.0, got %.6f", sims[1])
	}
	if ts[1] != 2.0 {
		t.Errorf("pair 1-2 timestamp: expected 2.0, got %.1f", ts[1])
	}
}

func TestFrameVectorWriter_Rollback(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recID = RecordingID(2)

	w, err := NewFrameVectorWriter(recID, testDim)
	if err != nil {
		t.Fatalf("NewFrameVectorWriter: %v", err)
	}
	if err := w.Write(unitVec(0), 0.0); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.Rollback()

	ts, sims, err := QueryConsecutiveSimilarities(recID)
	if err != nil {
		t.Fatalf("QueryConsecutiveSimilarities: %v", err)
	}
	if len(sims) != 0 || len(ts) != 0 {
		t.Errorf("expected no results after rollback, got %d similarities", len(sims))
	}
}

func TestDeleteFrameVectorsByRecordingID(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recID = RecordingID(3)
	vecs := [][]float32{unitVec(0), unitVec(1), unitVec(2)}
	timestamps := []float64{0.0, 1.0, 2.0}

	w, err := NewFrameVectorWriter(recID, testDim)
	if err != nil {
		t.Fatalf("NewFrameVectorWriter: %v", err)
	}
	for i, v := range vecs {
		w.Write(v, timestamps[i])
	}
	w.Commit()

	// Verify rows were inserted.
	_, sims, _ := QueryConsecutiveSimilarities(recID)
	if len(sims) != 2 {
		t.Fatalf("expected 2 similarities before delete, got %d", len(sims))
	}

	if err := DeleteFrameVectorsByRecordingID(recID); err != nil {
		t.Fatalf("DeleteFrameVectorsByRecordingID: %v", err)
	}

	_, sims, _ = QueryConsecutiveSimilarities(recID)
	if len(sims) != 0 {
		t.Errorf("expected 0 similarities after delete, got %d", len(sims))
	}
}

func TestSearchSimilarFrames(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recID = RecordingID(4)
	vecs := [][]float32{unitVec(0), unitVec(1), unitVec(2)}
	w, _ := NewFrameVectorWriter(recID, testDim)
	for i, v := range vecs {
		w.Write(v, float64(i))
	}
	w.Commit()

	// Query closest to unitVec(0); should return row for timestamp 0.
	results, err := SearchSimilarFrames(unitVec(0), 1)
	if err != nil {
		t.Fatalf("SearchSimilarFrames: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Timestamp != 0.0 {
		t.Errorf("expected timestamp 0.0, got %.1f", results[0].Timestamp)
	}
}

func TestSearchSimilarFramesByRecording_Isolation(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	// Two recordings with different vectors; query should only return recA results.
	const recA = RecordingID(5)
	const recB = RecordingID(6)

	wA, _ := NewFrameVectorWriter(recA, testDim)
	wA.Write(unitVec(0), 10.0)
	wA.Commit()

	wB, _ := NewFrameVectorWriter(recB, testDim)
	wB.Write(unitVec(15), 20.0)
	wB.Commit()

	results, err := SearchSimilarFramesByRecording(recA, unitVec(0), 3)
	if err != nil {
		t.Fatalf("SearchSimilarFramesByRecording: %v", err)
	}
	for _, r := range results {
		if r.RecordingID != recA {
			t.Errorf("result belongs to recording %d, expected %d", r.RecordingID, recA)
		}
	}
}

func TestSearchSimilarRecordingsByVector(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recA = RecordingID(7)
	const recB = RecordingID(8)

	wA, _ := NewFrameVectorWriter(recA, testDim)
	wA.Write(unitVec(0), 10.0)
	wA.Commit()

	wB, _ := NewFrameVectorWriter(recB, testDim)
	wB.Write(unitVec(1), 20.0)
	wB.Commit()

	// Test 1: Ensure it finds the exact match correctly with high k internally
	results, err := SearchSimilarRecordingsByVector(unitVec(0), 0.8, 10)
	if err != nil {
		t.Fatalf("SearchSimilarRecordingsByVector: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 recording match, got %d", len(results))
	}
	if results[0].RecordingID != recA {
		t.Fatalf("expected recA (%d), got %d", recA, results[0].RecordingID)
	}
	if math.Abs(results[0].Similarity-1.0) > 0.001 {
		t.Fatalf("expected similarity ~1.0, got %.4f", results[0].Similarity)
	}

	// Test 2: Regression test to ensure the max k value doesn't cause a failure
	// We call the query directly to guarantee it doesn't return an error from the sqlite-vec extension.
	// If the k-limit is violated, err will be non-nil.
	_, err = SearchSimilarRecordingsByVector(unitVec(0), 0.0, 50)
	if err != nil {
		t.Fatalf("SearchSimilarRecordingsByVector returned error on wide search (regression): %v", err)
	}
}

func TestQueryRecordingSimilarityEdges(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	const recA = RecordingID(9)
	const recB = RecordingID(10)
	const recC = RecordingID(11)

	wA, _ := NewFrameVectorWriter(recA, testDim)
	wA.Write(unitVec(0), 1.0)
	wA.Commit()

	wB, _ := NewFrameVectorWriter(recB, testDim)
	wB.Write(unitVec(0), 2.0)
	wB.Commit()

	wC, _ := NewFrameVectorWriter(recC, testDim)
	wC.Write(unitVec(3), 3.0)
	wC.Commit()

	edges, err := QueryRecordingSimilarityEdges(0.9, []RecordingID{recA, recB, recC}, 100)
	if err != nil {
		t.Fatalf("QueryRecordingSimilarityEdges: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected exactly 1 strong edge, got %d", len(edges))
	}

	ab := (edges[0].RecordingA == recA && edges[0].RecordingB == recB) ||
		(edges[0].RecordingA == recB && edges[0].RecordingB == recA)
	if !ab {
		t.Fatalf("expected edge between recA and recB, got %+v", edges[0])
	}
}

// TestNilWriter ensures a nil FrameVectorWriter is safe to call on all methods.
func TestNilWriter(t *testing.T) {
	var w *FrameVectorWriter
	if err := w.Write([]float32{1, 2, 3}, 0.0); err != nil {
		t.Errorf("nil Write returned error: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Errorf("nil Commit returned error: %v", err)
	}
	w.Rollback() // must not panic
}

// TestSerializeFloat32 verifies the IEEE-754 little-endian serialisation.
func TestSerializeFloat32(t *testing.T) {
	blob := serializeFloat32([]float32{1.0, 2.0, 3.0})
	if len(blob) != 12 {
		t.Fatalf("expected 12 bytes, got %d", len(blob))
	}
	// 1.0 as IEEE-754 LE bytes = 00 00 80 3F
	if blob[0] != 0x00 || blob[1] != 0x00 || blob[2] != 0x80 || blob[3] != 0x3F {
		t.Errorf("1.0 serialised incorrectly: %x", blob[:4])
	}
}
