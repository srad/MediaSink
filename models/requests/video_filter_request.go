package requests

import (
	"github.com/srad/mediasink/queries"
)

type VideoSortColumn string

const SortColumnCreatedAt VideoSortColumn = "created_at"
const SortColumnSize VideoSortColumn = "size"
const SortColumnDuration VideoSortColumn = "duration"

type VideoFilterRequest struct {
	Skip       int               `json:"skip"`
	Take       int               `json:"take"`
	SortOrder  queries.SortOrder `json:"sortOrder" extensions:"!x-nullable"`
	SortColumn VideoSortColumn   `json:"sortColumn" extensions:"!x-nullable"`
}

func (so VideoSortColumn) String() string {
	return string(so)
}

func (so VideoSortColumn) IsValid() bool {
	switch so {
	case SortColumnCreatedAt, SortColumnSize, SortColumnDuration:
		return true
	}
	return false
}
