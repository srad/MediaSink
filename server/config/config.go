package config

import (
	"fmt"
	"os"
	"runtime"
	"strings"
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

	// JWTSecret signs and validates bearer tokens. Held here rather than read per
	// request: the middleware used to call os.Getenv("SECRET") on every
	// authenticated request, which also made it untestable without mutating the
	// process environment.
	JWTSecret string

	// Database connection settings. DBAdapter selects the driver; the remaining
	// fields build the DSN for the non-SQLite drivers.
	DBAdapter  string
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBPort     string

	// ONNXRuntimeLib optionally overrides the shared library path the ONNX runtime
	// loads. Empty means "use the system default".
	ONNXRuntimeLib string
}

var (
	once            sync.Once
	cached          Cfg
	cachedWarnings  []string
	cachedErr       error
	requiredEnvVars = []string{"DB_FILENAME", "REC_PATH", "DATA_DIR", "DATA_DISK", "NET_ADAPTER", "SECRET"}
)

// Parse builds a Cfg from getenv. It is pure: it touches neither the process
// environment nor the logger, which is what makes it unit-testable.
//
// Warnings are returned rather than logged because Parse necessarily runs before
// log.SetLevel — logging here would emit at logrus' default level and ignore
// LOG_LEVEL. The composition root sets the level, then emits them.
//
// Every missing required variable is reported in one error rather than failing on
// the first, so a misconfigured deployment learns everything it is missing at once.
func Parse(getenv func(string) string) (Cfg, []string, error) {
	var warnings []string

	logLevel, logErr := ParseLogLevel(getenv("LOG_LEVEL"))
	if logErr != nil {
		warnings = append(warnings, fmt.Sprintf("invalid LOG_LEVEL %q, defaulting to %q: %v", getenv("LOG_LEVEL"), logLevel, logErr))
	}

	streamDebugLevel, streamDebugErr := ParseStreamDebugLevel(getenv("STREAM_DEBUG_LEVEL"))
	if streamDebugErr != nil {
		warnings = append(warnings, fmt.Sprintf("invalid STREAM_DEBUG_LEVEL %q, defaulting to %q: %v", getenv("STREAM_DEBUG_LEVEL"), streamDebugLevel, streamDebugErr))
	}

	cfg := Cfg{
		DbFileName:             getenv("DB_FILENAME"),
		RecordingsAbsolutePath: getenv("REC_PATH"),
		DataPath:               getenv("DATA_DIR"),
		DataDisk:               getenv("DATA_DISK"),
		NetworkDev:             getenv("NET_ADAPTER"),
		LogLevel:               logLevel,
		StreamDebugProfile:     NewStreamDebugProfile(streamDebugLevel),

		JWTSecret: getenv("SECRET"),

		DBAdapter:  getenv("DB_ADAPTER"),
		DBHost:     getenv("DB_HOST"),
		DBUser:     getenv("DB_USER"),
		DBPassword: getenv("DB_PASSWORD"),
		DBName:     getenv("DB_NAME"),
		DBPort:     getenv("DB_PORT"),

		ONNXRuntimeLib: getenv("ONNXRUNTIME_LIB"),
	}

	var missing []string
	for _, key := range requiredEnvVars {
		if getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return cfg, warnings, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, warnings, nil
}

// Load reads the process environment once and caches the result, warnings and error
// alike, so every caller sees the same configuration.
func Load() (Cfg, []string, error) {
	once.Do(func() {
		cached, cachedWarnings, cachedErr = Parse(os.Getenv)
	})
	return cached, cachedWarnings, cachedErr
}

// Read is the panicking shim retained only for the call sites in internal/db and
// internal/services that still reach for configuration directly; Phases 2b and 3 of
// the refactor remove them, and this function goes with the last one. Do not add
// callers — take a Cfg parameter instead.
//
// It deliberately drops the warnings: Load caches them, so re-emitting here would
// repeat the same warning on every call, and the composition root has already
// logged them once at the configured level.
func Read() Cfg {
	cfg, _, err := Load()
	if err != nil {
		log.Panicf("configuration error: %v", err)
	}
	return cfg
}
