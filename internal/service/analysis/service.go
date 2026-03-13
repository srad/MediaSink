package analysis

import (
	"context"

	"github.com/srad/mediasink/internal/db"
	legacysvc "github.com/srad/mediasink/internal/services"
	"github.com/srad/mediasink/internal/store"
)

type Service struct {
	recordings store.RecordingStore
	jobs       store.JobStore
	analyses   store.AnalysisStore
}

func NewService(recordings store.RecordingStore, jobs store.JobStore, analyses store.AnalysisStore) *Service {
	return &Service{
		recordings: recordings,
		jobs:       jobs,
		analyses:   analyses,
	}
}

func (s *Service) CreateJob(ctx context.Context, recordingID db.RecordingID) (*db.Job, legacysvc.PreviewValidationResult, error) {
	recording, err := s.recordings.FindByID(ctx, recordingID)
	if err != nil {
		return nil, legacysvc.PreviewValidationResult{}, err
	}

	previewState, err := legacysvc.ValidateRecordingPreview(recording)
	if err != nil && !previewState.NeedsRegeneration {
		return nil, previewState, err
	}
	if previewState.NeedsRegeneration {
		return nil, previewState, nil
	}

	job, err := s.jobs.EnqueueAnalysis(ctx, recording)
	return job, previewState, err
}

func (s *Service) GetResult(ctx context.Context, recordingID db.RecordingID) (*db.VideoAnalysisResult, error) {
	return s.analyses.FindByRecordingID(ctx, recordingID)
}
