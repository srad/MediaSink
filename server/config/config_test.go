package config

import (
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
)

// fakeEnv turns a map into the getenv function Parse takes. Every test here drives
// Parse through one of these, so the suite never touches the process environment —
// which is the whole point of Parse being a parameterised function.
func fakeEnv(vars map[string]string) func(string) string {
	return func(key string) string { return vars[key] }
}

// completeEnv is the minimum that parses without error, plus the optional settings.
func completeEnv() map[string]string {
	return map[string]string{
		"DB_FILENAME":        "/data/mediasink.db",
		"REC_PATH":           "/recordings",
		"DATA_DIR":           ".previews",
		"DATA_DISK":          "/",
		"NET_ADAPTER":        "eth0",
		"SECRET":             "jwt-signing-secret",
		"LOG_LEVEL":          "debug",
		"STREAM_DEBUG_LEVEL": "trace",
		"DB_ADAPTER":         "postgres",
		"DB_HOST":            "db.internal",
		"DB_USER":            "mediasink",
		"DB_PASSWORD":        "hunter2",
		"DB_NAME":            "mediasink_prod",
		"DB_PORT":            "5432",
		"ONNXRUNTIME_LIB":    "/opt/onnx/libonnxruntime.so",
	}
}

func TestParseCompleteEnvironment(t *testing.T) {
	cfg, warnings, err := Parse(fakeEnv(completeEnv()))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Parse() warnings = %v, want none", warnings)
	}

	tests := []struct {
		field string
		got   string
		want  string
	}{
		{"DbFileName", cfg.DbFileName, "/data/mediasink.db"},
		{"RecordingsAbsolutePath", cfg.RecordingsAbsolutePath, "/recordings"},
		{"DataPath", cfg.DataPath, ".previews"},
		{"DataDisk", cfg.DataDisk, "/"},
		{"NetworkDev", cfg.NetworkDev, "eth0"},
		{"JWTSecret", cfg.JWTSecret, "jwt-signing-secret"},
		{"DBAdapter", cfg.DBAdapter, "postgres"},
		{"DBHost", cfg.DBHost, "db.internal"},
		{"DBUser", cfg.DBUser, "mediasink"},
		{"DBPassword", cfg.DBPassword, "hunter2"},
		{"DBName", cfg.DBName, "mediasink_prod"},
		{"DBPort", cfg.DBPort, "5432"},
		{"ONNXRuntimeLib", cfg.ONNXRuntimeLib, "/opt/onnx/libonnxruntime.so"},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("cfg.%s = %q, want %q", test.field, test.got, test.want)
		}
	}

	if cfg.LogLevel != log.DebugLevel {
		t.Errorf("cfg.LogLevel = %v, want %v", cfg.LogLevel, log.DebugLevel)
	}
	if cfg.StreamDebugProfile == nil {
		t.Fatal("cfg.StreamDebugProfile = nil, want a profile")
	}
	if got := cfg.StreamDebugProfile.Level(); got != StreamDebugTrace {
		t.Errorf("cfg.StreamDebugProfile.Level() = %q, want %q", got, StreamDebugTrace)
	}
}

// Each required variable must be reported on its own, so a deployment missing one
// is told which one.
func TestParseReportsEachMissingRequiredVar(t *testing.T) {
	for _, key := range requiredEnvVars {
		t.Run(key, func(t *testing.T) {
			vars := completeEnv()
			delete(vars, key)

			_, _, err := Parse(fakeEnv(vars))
			if err == nil {
				t.Fatalf("Parse() without %s: error = nil, want an error", key)
			}
			if !strings.Contains(err.Error(), key) {
				t.Errorf("Parse() error = %q, want it to name %s", err, key)
			}
			// Only the missing one should be named.
			for _, other := range requiredEnvVars {
				if other != key && strings.Contains(err.Error(), other) {
					t.Errorf("Parse() error = %q, unexpectedly names %s", err, other)
				}
			}
		})
	}
}

// Failing on the first missing variable would make a misconfigured deployment fix
// one, restart, and discover the next. Parse reports them together.
func TestParseReportsAllMissingRequiredVarsAtOnce(t *testing.T) {
	_, _, err := Parse(fakeEnv(nil))
	if err == nil {
		t.Fatal("Parse() with empty environment: error = nil, want an error")
	}

	want := "missing required environment variables: " + strings.Join(requiredEnvVars, ", ")
	if err.Error() != want {
		t.Errorf("Parse() error = %q, want %q", err, want)
	}
}

// A config that fails still comes back populated, so a caller that logs the error
// can also report what it did manage to read.
func TestParseReturnsPartialConfigOnError(t *testing.T) {
	vars := completeEnv()
	delete(vars, "SECRET")

	cfg, _, err := Parse(fakeEnv(vars))
	if err == nil {
		t.Fatal("Parse() error = nil, want an error")
	}
	if cfg.DbFileName != "/data/mediasink.db" {
		t.Errorf("cfg.DbFileName = %q, want the parsed value even on error", cfg.DbFileName)
	}
	if cfg.JWTSecret != "" {
		t.Errorf("cfg.JWTSecret = %q, want empty", cfg.JWTSecret)
	}
}

func TestParseWarnsButDoesNotFailOnInvalidLevels(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		wantWarning string
		check       func(t *testing.T, cfg Cfg)
	}{
		{
			name:        "invalid LOG_LEVEL falls back to info",
			key:         "LOG_LEVEL",
			value:       "louder",
			wantWarning: "invalid LOG_LEVEL",
			check: func(t *testing.T, cfg Cfg) {
				if cfg.LogLevel != log.InfoLevel {
					t.Errorf("cfg.LogLevel = %v, want %v", cfg.LogLevel, log.InfoLevel)
				}
			},
		},
		{
			name:        "invalid STREAM_DEBUG_LEVEL falls back to error",
			key:         "STREAM_DEBUG_LEVEL",
			value:       "verbose",
			wantWarning: "invalid STREAM_DEBUG_LEVEL",
			check: func(t *testing.T, cfg Cfg) {
				if got := cfg.StreamDebugProfile.Level(); got != StreamDebugError {
					t.Errorf("cfg.StreamDebugProfile.Level() = %q, want %q", got, StreamDebugError)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vars := completeEnv()
			vars[test.key] = test.value

			cfg, warnings, err := Parse(fakeEnv(vars))
			if err != nil {
				t.Fatalf("Parse() error = %v, want nil — a bad level is a warning, not a failure", err)
			}
			if len(warnings) != 1 {
				t.Fatalf("Parse() warnings = %v, want exactly 1", warnings)
			}
			if !strings.Contains(warnings[0], test.wantWarning) {
				t.Errorf("warning = %q, want it to contain %q", warnings[0], test.wantWarning)
			}
			test.check(t, cfg)
		})
	}
}

// Unset is not invalid: both levels have defaults, and neither should warn.
func TestParseDoesNotWarnOnUnsetOptionalLevels(t *testing.T) {
	vars := completeEnv()
	delete(vars, "LOG_LEVEL")
	delete(vars, "STREAM_DEBUG_LEVEL")

	cfg, warnings, err := Parse(fakeEnv(vars))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(warnings) != 0 {
		t.Errorf("Parse() warnings = %v, want none", warnings)
	}
	if cfg.LogLevel != log.InfoLevel {
		t.Errorf("cfg.LogLevel = %v, want %v", cfg.LogLevel, log.InfoLevel)
	}
	if got := cfg.StreamDebugProfile.Level(); got != StreamDebugError {
		t.Errorf("cfg.StreamDebugProfile.Level() = %q, want %q", got, StreamDebugError)
	}
}

// Both bad at once produce both warnings, in the order Parse evaluates them.
func TestParseAccumulatesMultipleWarnings(t *testing.T) {
	vars := completeEnv()
	vars["LOG_LEVEL"] = "louder"
	vars["STREAM_DEBUG_LEVEL"] = "verbose"

	_, warnings, err := Parse(fakeEnv(vars))
	if err != nil {
		t.Fatalf("Parse() error = %v, want nil", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("Parse() warnings = %v, want 2", warnings)
	}
	if !strings.Contains(warnings[0], "LOG_LEVEL") || !strings.Contains(warnings[1], "STREAM_DEBUG_LEVEL") {
		t.Errorf("warnings = %v, want LOG_LEVEL first then STREAM_DEBUG_LEVEL", warnings)
	}
}
