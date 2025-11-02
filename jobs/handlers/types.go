package handlers

import "github.com/srad/mediasink/database"

// JobMessage is a generic message wrapper for job-related events
type JobMessage[T any] struct {
	Job  *database.Job `json:"job"`
	Data T             `json:"data"`
}
