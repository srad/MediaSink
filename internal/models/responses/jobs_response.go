package responses

import "github.com/srad/mediasink/internal/db"

type JobsResponse struct {
	Jobs       []*db.Job `json:"jobs"`
	TotalCount int64           `json:"totalCount" extensions:"!x-nullable"`
	Skip       int             `json:"skip"  extensions:"!x-nullable"`
	Take       int             `json:"take"  extensions:"!x-nullable"`
}
