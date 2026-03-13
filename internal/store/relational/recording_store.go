package relational

import (
	"context"

	"github.com/srad/mediasink/internal/db"
)

type RecordingStore struct{}

func NewRecordingStore() *RecordingStore {
	return &RecordingStore{}
}

func (s *RecordingStore) List(_ context.Context) ([]*db.Recording, error) {
	return db.RecordingsList()
}

func (s *RecordingStore) FindByID(_ context.Context, recordingID db.RecordingID) (*db.Recording, error) {
	return db.FindRecordingByID(recordingID)
}

func (s *RecordingStore) FindByIDs(_ context.Context, ids []db.RecordingID) ([]*db.Recording, error) {
	return db.FindRecordingsByIDs(ids)
}
