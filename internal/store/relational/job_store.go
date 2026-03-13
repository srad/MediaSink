package relational

import (
	"context"

	"github.com/srad/mediasink/internal/db"
)

type JobStore struct{}

func NewJobStore() *JobStore {
	return &JobStore{}
}

func (s *JobStore) List(_ context.Context, skip, take int, statuses []db.JobStatus, order db.JobOrder) ([]*db.Job, int64, error) {
	return db.JobList(skip, take, statuses, order)
}

func (s *JobStore) FindByID(_ context.Context, id uint) (*db.Job, error) {
	var job db.Job
	if err := db.DB.
		Preload("Channel").
		Preload("Recording").
		Where("job_id = ?", id).
		First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *JobStore) EnqueuePreview(_ context.Context, recording *db.Recording) (*db.Job, error) {
	return recording.EnqueuePreviewFramesJob()
}

func (s *JobStore) EnqueueAnalysis(_ context.Context, recording *db.Recording) (*db.Job, error) {
	return recording.EnqueueAnalysisJob()
}
