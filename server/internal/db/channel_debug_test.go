package db

import (
	"reflect"
	"testing"

	"github.com/srad/mediasink/server/config"
)

func TestBuildQueryStreamURLArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		level    config.StreamDebugLevel
		url      string
		expected []string
	}{
		{
			name:  "default error output",
			level: config.StreamDebugError,
			url:   "https://example.com/live",
			expected: []string{
				"--force-ipv4",
				"--no-warnings",
				"-f", "bv*+ba/b",
				"--get-url",
				"https://example.com/live",
			},
		},
		{
			name:  "trace includes traffic logging",
			level: config.StreamDebugTrace,
			url:   "https://example.com/live",
			expected: []string{
				"--force-ipv4",
				"--verbose",
				"--print-traffic",
				"-f", "bv*+ba/b",
				"--get-url",
				"https://example.com/live",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			args := buildQueryStreamURLArgs(test.url, config.NewStreamDebugProfile(test.level))
			if !reflect.DeepEqual(args, test.expected) {
				t.Fatalf("expected args %v, got %v", test.expected, args)
			}
		})
	}
}

func TestExtractStreamURLs(t *testing.T) {
	t.Parallel()

	output := "https://video.example/live.m3u8\nhttps://audio.example/live.m3u8\n"
	expected := []string{
		"https://video.example/live.m3u8",
		"https://audio.example/live.m3u8",
	}

	if got := extractStreamURLs(output); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected urls %v, got %v", expected, got)
	}
}
