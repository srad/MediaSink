package vector

import (
	"context"
	"fmt"
	"sync"

	"github.com/srad/mediasink/server/internal/db"
)

type Embedding struct {
	Values    []float32
	Timestamp float64
}

type Store interface {
	Initialize(context.Context) error
	DeleteRecording(context.Context, db.RecordingID) error
	WriteEmbeddings(context.Context, db.RecordingID, []Embedding) error
	QueryConsecutiveSimilarities(context.Context, db.RecordingID) ([]float64, []float64, error)
	SearchSimilarFrames(context.Context, []float32, int) ([]db.FrameVectorResult, error)
	SearchSimilarFramesByRecording(context.Context, db.RecordingID, []float32, int) ([]db.FrameVectorResult, error)
	SearchSimilarRecordings(context.Context, []float32, float64, int) ([]db.SimilarRecordingResult, error)
	ListRecordingIDs(context.Context, int) ([]db.RecordingID, error)
	QueryRecordingSimilarityEdges(context.Context, float64, []db.RecordingID, int) ([]db.RecordingSimilarityEdge, error)
}

type SQLiteVecStore struct {
	// adapter is the configured database driver. A plain string rather than a
	// config.Cfg so this package keeps no dependency on config.
	adapter string
}

var (
	defaultStore Store
	defaultMu    sync.RWMutex
)

func NewSQLiteVecStore(adapter string) *SQLiteVecStore {
	return &SQLiteVecStore{adapter: adapter}
}

func SetDefault(store Store) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultStore = store
}

func Default() Store {
	defaultMu.RLock()
	store := defaultStore
	defaultMu.RUnlock()
	if store != nil {
		return store
	}

	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultStore == nil {
		// Empty adapter means sqlite, the only backend this store supports. The
		// fallback is unreachable in the shipped binary — app.InitializeApp always
		// calls SetDefault first — and disappears with Default() itself in Phase 3.
		defaultStore = NewSQLiteVecStore("")
	}
	return defaultStore
}

func (s *SQLiteVecStore) Initialize(ctx context.Context) error {
	if s.adapter != "" && s.adapter != "sqlite" && s.adapter != "sqlite3" {
		return fmt.Errorf("sqlite-vec backend requires sqlite, current adapter is %q", s.adapter)
	}
	if db.DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var version string
	if err := sqlDB.QueryRowContext(ctx, `SELECT vec_version()`).Scan(&version); err != nil {
		return fmt.Errorf("sqlite-vec not available: %w", err)
	}
	return nil
}

func (s *SQLiteVecStore) DeleteRecording(_ context.Context, recordingID db.RecordingID) error {
	return db.DeleteFrameVectorsByRecordingID(recordingID)
}

func (s *SQLiteVecStore) WriteEmbeddings(_ context.Context, recordingID db.RecordingID, embeddings []Embedding) error {
	if len(embeddings) == 0 {
		return nil
	}

	writer, err := db.NewFrameVectorWriter(recordingID, len(embeddings[0].Values))
	if err != nil {
		return err
	}

	for i, embedding := range embeddings {
		if err := writer.Write(embedding.Values, embedding.Timestamp); err != nil {
			writer.Rollback()
			return fmt.Errorf("write embedding %d: %w", i, err)
		}
	}

	if err := writer.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteVecStore) QueryConsecutiveSimilarities(_ context.Context, recordingID db.RecordingID) ([]float64, []float64, error) {
	return db.QueryConsecutiveSimilarities(recordingID)
}

func (s *SQLiteVecStore) SearchSimilarFrames(_ context.Context, queryVector []float32, limit int) ([]db.FrameVectorResult, error) {
	return db.SearchSimilarFrames(queryVector, limit)
}

func (s *SQLiteVecStore) SearchSimilarFramesByRecording(_ context.Context, recordingID db.RecordingID, queryVector []float32, limit int) ([]db.FrameVectorResult, error) {
	return db.SearchSimilarFramesByRecording(recordingID, queryVector, limit)
}

func (s *SQLiteVecStore) SearchSimilarRecordings(_ context.Context, queryVector []float32, minSimilarity float64, limit int) ([]db.SimilarRecordingResult, error) {
	return db.SearchSimilarRecordingsByVector(queryVector, minSimilarity, limit)
}

func (s *SQLiteVecStore) ListRecordingIDs(_ context.Context, limit int) ([]db.RecordingID, error) {
	return db.ListRecordingIDsWithFrameVectors(limit)
}

func (s *SQLiteVecStore) QueryRecordingSimilarityEdges(_ context.Context, minSimilarity float64, recordingIDs []db.RecordingID, limit int) ([]db.RecordingSimilarityEdge, error) {
	return db.QueryRecordingSimilarityEdges(minSimilarity, recordingIDs, limit)
}
