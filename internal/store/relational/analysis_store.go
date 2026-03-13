package relational

import (
	"context"

	"github.com/srad/mediasink/internal/db"
)

type AnalysisStore struct{}

func NewAnalysisStore() *AnalysisStore {
	return &AnalysisStore{}
}

func (s *AnalysisStore) FindByRecordingID(_ context.Context, recordingID db.RecordingID) (*db.VideoAnalysisResult, error) {
	return db.GetAnalysisByRecordingID(recordingID)
}
