package responses

import (
	"github.com/srad/mediasink/internal/db"
)

type VideoFilterResponse struct {
	Videos     []*db.Recording `json:"videos"`
	TotalCount int64                 `json:"totalCount" extensions:"!x-nullable"`
	Skip       int                   `json:"skip"  extensions:"!x-nullable"`
	Take       int                   `json:"take"  extensions:"!x-nullable"`
}
