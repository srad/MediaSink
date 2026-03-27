package config

import (
	"os"
	"runtime"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	winFont   = "C\\\\:/Windows/Fonts/DMMono-Regular.ttf"
	linuxFont = "/usr/share/fonts/truetype/DMMono-Regular.ttf"

	// Preview frame settings
	FrameHeight = 224 // Maximum height for preview frames, width scaled proportionally
)

var (
	ThreadCount        = uint(float32(runtime.NumCPU() / 2))
	FastJobThreadCount = max(1, runtime.NumCPU()/3)
	SlowJobThreadCount = max(1, runtime.NumCPU()/3)
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Cfg struct {
	DbFileName             string
	RecordingsAbsolutePath string
	DataDisk               string
	NetworkDev             string
	DataPath               string
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
		cached = Cfg{
			DbFileName:             mustEnv("DB_FILENAME"),
			RecordingsAbsolutePath: mustEnv("REC_PATH"),
			DataPath:               mustEnv("DATA_DIR"),
			DataDisk:               mustEnv("DATA_DISK"),
			NetworkDev:             mustEnv("NET_ADAPTER"),
		}
	})
	return cached
}

func GetFontPath() string {
	if runtime.GOOS == "windows" {
		return winFont
	}
	return linuxFont
}
