package db

import (
	"os"
	"testing"
)

// TestMain sets the minimal environment variables required by config.Read() so
// that database package tests don't panic when running without a full config.
func TestMain(m *testing.M) {
	setIfEmpty := func(key, val string) {
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	setIfEmpty("DB_FILENAME", ":memory:")
	setIfEmpty("REC_PATH", "/tmp")
	setIfEmpty("DATA_DIR", ".previews")
	setIfEmpty("DATA_DISK", "/")
	setIfEmpty("NET_ADAPTER", "eth0")

	os.Exit(m.Run())
}
