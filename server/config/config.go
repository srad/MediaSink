package config

import (
	"os"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	// Preview frame settings
	FrameHeight = 224 // Maximum height for preview frames, width scaled proportionally
)

var (
	ThreadCount        = uint(float32(runtime.NumCPU() / 2))
	FastJobThreadCount = max(1, runtime.NumCPU()/3)
	SlowJobThreadCount = max(1, runtime.NumCPU()/3)
)

// max was a hand-rolled shadow of the builtin added in Go 1.21; the builtin has
// identical semantics for these int arguments.

type Cfg struct {
	DbFileName             string
	RecordingsAbsolutePath string
	DataDisk               string
	NetworkDev             string
	DataPath               string
	LogLevel               log.Level
	StreamDebugProfile     *StreamDebugProfile
}

var (
	once   sync.Once
	cached Cfg
)

func mustEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Panicf("required environment variable %s is not set", key)
	}
	return val
}

func Read() Cfg {
	once.Do(func() {
		streamDebugLevel, streamDebugErr := ParseStreamDebugLevel(os.Getenv("STREAM_DEBUG_LEVEL"))
		if streamDebugErr != nil {
			log.Warnf("invalid STREAM_DEBUG_LEVEL %q, defaulting to %q: %v", os.Getenv("STREAM_DEBUG_LEVEL"), streamDebugLevel, streamDebugErr)
		}

		cached = Cfg{
			DbFileName:             mustEnv("DB_FILENAME"),
			RecordingsAbsolutePath: mustEnv("REC_PATH"),
			DataPath:               mustEnv("DATA_DIR"),
			DataDisk:               mustEnv("DATA_DISK"),
			NetworkDev:             mustEnv("NET_ADAPTER"),
			LogLevel:               log.GetLevel(),
			StreamDebugProfile:     NewStreamDebugProfile(streamDebugLevel),
		}
	})
	return cached
}
