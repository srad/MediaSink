package services

import (
	"reflect"
	"testing"

	"github.com/srad/mediasink/server/config"
)

func TestBuildCaptureCommandArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    config.StreamDebugLevel
		skip     uint
		expected []string
	}{
		{
			name:  "default error level",
			level: config.StreamDebugError,
			skip:  5,
			expected: []string{
				"-hide_banner",
				"-loglevel", "error",
				"-i", "https://stream.test/live",
				"-ss", "5",
				"-movflags", "faststart",
				"-c", "copy",
				"/tmp/output.mp4",
			},
		},
		{
			name:  "trace level",
			level: config.StreamDebugTrace,
			skip:  0,
			expected: []string{
				"-hide_banner",
				"-loglevel", "trace",
				"-i", "https://stream.test/live",
				"-ss", "0",
				"-movflags", "faststart",
				"-c", "copy",
				"/tmp/output.mp4",
			},
		},
		{
			name:  "split video audio inputs",
			level: config.StreamDebugDebug,
			skip:  2,
			expected: []string{
				"-hide_banner",
				"-loglevel", "debug",
				"-i", "https://stream.test/live",
				"-i", "https://stream.test/audio",
				"-ss", "2",
				"-map", "0:v:0",
				"-map", "1:a:0",
				"-movflags", "faststart",
				"-c", "copy",
				"/tmp/output.mp4",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			inputURLs := []string{"https://stream.test/live"}
			if test.name == "split video audio inputs" {
				inputURLs = []string{"https://stream.test/live", "https://stream.test/audio"}
			}
			args := buildCaptureCommandArgs(inputURLs, test.skip, "/tmp/output.mp4", config.NewStreamDebugProfile(test.level))
			if !reflect.DeepEqual(args, test.expected) {
				t.Fatalf("expected args %v, got %v", test.expected, args)
			}
		})
	}
}
