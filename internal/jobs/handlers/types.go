package handlers

import "github.com/srad/mediasink/internal/db"

// JobMessage is a generic message wrapper for job-related events
type JobMessage[T any] struct {
	Job  *db.Job `json:"job"`
	Data T             `json:"data"`
}
