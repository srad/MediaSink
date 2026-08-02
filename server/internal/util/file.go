package util

import (
	"errors"
	"io/fs"
	"os"
)

// FileExists reports whether the named file or directory exists.
// Stat errors other than "not exist" (for example a permission error) are
// treated as existing, since the path could not be ruled out.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		return !errors.Is(err, fs.ErrNotExist)
	}
	return true
}
