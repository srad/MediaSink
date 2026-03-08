package requests

import "github.com/srad/mediasink/internal/db"

type JobsRequest struct {
	Skip      int                  `json:"skip"`
	Take      int                  `json:"take"`
	States    []db.JobStatus `json:"states" extensions:"!x-nullable"`
	SortOrder db.JobOrder    `json:"sortOrder" extensions:"!x-nullable"`
}
