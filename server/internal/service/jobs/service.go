package jobs

import (
	"context"

	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/store"
)

type Service struct {
	jobs store.JobStore
}

func NewService(jobs store.JobStore) *Service {
	return &Service{jobs: jobs}
}

func (s *Service) List(ctx context.Context, skip, take int, statuses []db.JobStatus, order db.JobOrder) ([]*db.Job, int64, error) {
	return s.jobs.List(ctx, skip, take, statuses, order)
}

func (s *Service) Get(ctx context.Context, jobID uint) (*db.Job, error) {
	return s.jobs.FindByID(ctx, jobID)
}
