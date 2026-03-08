package handlers

import (
	"github.com/srad/mediasink/internal/db"
)

// JobHandler defines the interface for all job handlers
type JobHandler interface {
	// Handle executes the job and returns an error if it fails
	Handle(job *db.Job, threadCount int) error
	// Name returns the name of this handler
	Name() string
}
