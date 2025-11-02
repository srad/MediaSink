package handlers

import (
	"github.com/srad/mediasink/database"
)

// JobHandler defines the interface for all job handlers
type JobHandler interface {
	// Handle executes the job and returns an error if it fails
	Handle(job *database.Job, threadCount int) error
	// Name returns the name of this handler
	Name() string
}
