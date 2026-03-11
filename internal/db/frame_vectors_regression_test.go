package db

import (
	"os"
	"strings"
	"testing"
)

// TestSearchSimilarRecordingsByVector_Regression_KLimit ensures that
// an overly large K value specifically causes a sqlite-vec error,
// verifying that our limit of 4000 is necessary.
func TestSearchSimilarRecordingsByVector_Regression_KLimit(t *testing.T) {
	if os.Getenv("DB_ADAPTER") != "" && os.Getenv("DB_ADAPTER") != "sqlite" {
		t.Skip("Skipping: sqlite-vec requires SQLite")
	}
	setupVecDB(t)

	// Write one frame so the table has data
	wA, _ := NewFrameVectorWriter(RecordingID(99), testDim)
	wA.Write(unitVec(0), 1.0)
	wA.Commit()

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get DB: %v", err)
	}

	// Deliberately test a k limit that is too high (4097+)
	emb := serializeFloat32(unitVec(0))
	rows, err := sqlDB.Query(`
		SELECT recording_id
		FROM frame_vectors
		WHERE embedding MATCH ? AND k = 5000
	`, emb)

	if err != nil {
		if !strings.Contains(err.Error(), "too large") && !strings.Contains(err.Error(), "limit") {
			t.Errorf("Expected error to mention limit or being too large, got: %v", err)
		}
		return // Expected error occurred on query
	}
	defer rows.Close()

	// The sqlite-vec error is lazily evaluated during rows.Next() or rows.Err()
	for rows.Next() {
	}

	err = rows.Err()
	if err == nil {
		t.Fatalf("Expected an error for k=5000 exceeding sqlite-vec's limit during row evaluation, but got none")
	}

	if !strings.Contains(err.Error(), "too large") && !strings.Contains(err.Error(), "limit") {
		t.Errorf("Expected error to mention limit or being too large, got: %v", err)
	}
}
