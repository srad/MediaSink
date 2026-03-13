package store

import (
	"context"

	"github.com/srad/mediasink/internal/db"
)

type UserStore interface {
	Create(context.Context, *db.User) error
	ExistsUsername(context.Context, string) (bool, error)
	FindByUsername(context.Context, string) (*db.User, error)
	FindByID(context.Context, uint) (*db.User, error)
}

type RecordingStore interface {
	List(context.Context) ([]*db.Recording, error)
	FindByID(context.Context, db.RecordingID) (*db.Recording, error)
	FindByIDs(context.Context, []db.RecordingID) ([]*db.Recording, error)
}

type JobStore interface {
	List(context.Context, int, int, []db.JobStatus, db.JobOrder) ([]*db.Job, int64, error)
	FindByID(context.Context, uint) (*db.Job, error)
	EnqueuePreview(context.Context, *db.Recording) (*db.Job, error)
	EnqueueAnalysis(context.Context, *db.Recording) (*db.Job, error)
}

type AnalysisStore interface {
	FindByRecordingID(context.Context, db.RecordingID) (*db.VideoAnalysisResult, error)
}
