package recordings

import (
	"context"

	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/store"
)

type Service struct {
	recordings store.RecordingStore
	jobs       store.JobStore
}

func NewService(recordings store.RecordingStore, jobs store.JobStore) *Service {
	return &Service{recordings: recordings, jobs: jobs}
}

func (s *Service) List(ctx context.Context) ([]*db.Recording, error) {
	return s.recordings.List(ctx)
}

func (s *Service) Get(ctx context.Context, id db.RecordingID) (*db.Recording, error) {
	return s.recordings.FindByID(ctx, id)
}

func (s *Service) CreatePreviewJob(ctx context.Context, id db.RecordingID) (*db.Job, error) {
	recording, err := s.recordings.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.jobs.EnqueuePreview(ctx, recording)
}
