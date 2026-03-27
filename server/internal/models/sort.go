package models

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

func (so SortOrder) IsValid() bool {
	switch so {
	case SortAsc, SortDesc:
		return true
	}
	return false
}

func (so SortOrder) String() string {
	return string(so)
}
