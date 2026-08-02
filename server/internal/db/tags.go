package db

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	rTags, _ = regexp.Compile(`^[a-z\-0-9]+(,[a-z\-0-9]+)*$`)
)

type Tags []string

// Scan restores tags from the database. A NULL column yields nil tags and an
// empty string yields an empty slice; splitting unconditionally would produce
// a single empty tag that Value() then refuses to write back.
func (o *Tags) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*o = nil
		return nil
	case string:
		if v == "" {
			*o = Tags{}
			return nil
		}
		*o = strings.Split(v, ",")
		return nil
	case []byte:
		if len(v) == 0 {
			*o = Tags{}
			return nil
		}
		*o = strings.Split(string(v), ",")
		return nil
	default:
		return errors.New("src value cannot cast to []string")
	}
}

func (o Tags) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	if err := o.IsValid(); err != nil {
		return nil, err
	}
	return strings.ToLower(strings.Join(o, ",")), nil
}

func (tags *Tags) IsValid() error {
	if tags == nil {
		return nil
	}
	for _, tag := range *tags {
		if !rTags.MatchString(tag) {
			return fmt.Errorf("invalid tag: %s", tag)
		}
	}
	return nil
}
