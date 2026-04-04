package config

import (
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestParseLogLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected log.Level
		wantErr  bool
	}{
		{name: "default", raw: "", expected: log.InfoLevel},
		{name: "debug", raw: "debug", expected: log.DebugLevel},
		{name: "warning alias", raw: "warning", expected: log.WarnLevel},
		{name: "trace uppercase", raw: "TRACE", expected: log.TraceLevel},
		{name: "invalid", raw: "chatty", expected: log.InfoLevel, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			level, err := ParseLogLevel(test.raw)
			if test.wantErr && err == nil {
				t.Fatalf("expected error for %q", test.raw)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", test.raw, err)
			}
			if level != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, level)
			}
		})
	}
}

func TestParseStreamDebugLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		expected StreamDebugLevel
		wantErr  bool
	}{
		{name: "default", raw: "", expected: StreamDebugError},
		{name: "warning", raw: "warning", expected: StreamDebugWarning},
		{name: "trace uppercase", raw: "TRACE", expected: StreamDebugTrace},
		{name: "invalid", raw: "noisy", expected: StreamDebugError, wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			level, err := ParseStreamDebugLevel(test.raw)
			if test.wantErr && err == nil {
				t.Fatalf("expected error for %q", test.raw)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error for %q: %v", test.raw, err)
			}
			if level != test.expected {
				t.Fatalf("expected %q, got %q", test.expected, level)
			}
		})
	}
}

func TestStreamDebugProfileMappings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		level        StreamDebugLevel
		ffmpegLevel  string
		ytDLPArgs    []string
		logDetails   bool
		logOnSuccess bool
	}{
		{
			name:        "quiet",
			level:       StreamDebugQuiet,
			ffmpegLevel: "quiet",
			ytDLPArgs:   []string{"--quiet", "--no-warnings"},
		},
		{
			name:        "error",
			level:       StreamDebugError,
			ffmpegLevel: "error",
			ytDLPArgs:   []string{"--no-warnings"},
		},
		{
			name:        "warning",
			level:       StreamDebugWarning,
			ffmpegLevel: "warning",
			ytDLPArgs:   nil,
		},
		{
			name:         "info",
			level:        StreamDebugInfo,
			ffmpegLevel:  "info",
			ytDLPArgs:    nil,
			logOnSuccess: true,
		},
		{
			name:         "debug",
			level:        StreamDebugDebug,
			ffmpegLevel:  "debug",
			ytDLPArgs:    []string{"--verbose"},
			logDetails:   true,
			logOnSuccess: true,
		},
		{
			name:         "trace",
			level:        StreamDebugTrace,
			ffmpegLevel:  "trace",
			ytDLPArgs:    []string{"--verbose", "--print-traffic"},
			logDetails:   true,
			logOnSuccess: true,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			profile := NewStreamDebugProfile(test.level)

			if got := profile.FFmpegLogLevel(); got != test.ffmpegLevel {
				t.Fatalf("expected ffmpeg log level %q, got %q", test.ffmpegLevel, got)
			}
			if got := profile.YTDLPArgs(); !reflect.DeepEqual(got, test.ytDLPArgs) {
				t.Fatalf("expected yt-dlp args %v, got %v", test.ytDLPArgs, got)
			}
			if got := profile.LogCommandDetails(); got != test.logDetails {
				t.Fatalf("expected LogCommandDetails=%t, got %t", test.logDetails, got)
			}
			if got := profile.LogCommandOutputOnSuccess(); got != test.logOnSuccess {
				t.Fatalf("expected LogCommandOutputOnSuccess=%t, got %t", test.logOnSuccess, got)
			}
		})
	}
}
