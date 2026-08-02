package util

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConversionMediaTypeValidate(t *testing.T) {
	cases := []struct {
		mediaType ConversionMediaType
		want      bool
	}{
		{Convert720p, true},
		{Convert1080p, true},
		{"mp3", false},
		{"999", false},
		{"720p", false},
		{"", false},
	}

	for _, tc := range cases {
		if got := tc.mediaType.Validate(); got != tc.want {
			t.Errorf("ConversionMediaType(%q).Validate() = %v, want %v", tc.mediaType, got, tc.want)
		}
	}
}

// ConvertVideo must reject unsupported media types before touching the file
// system or spawning ffmpeg. "mp3" is explicitly no longer supported.
func TestConvertVideoRejectsUnsupportedMediaType(t *testing.T) {
	for _, mediaType := range []ConversionMediaType{"mp3", "999", "720p", ""} {
		args := &VideoConversionArgs{
			// Deliberately bogus: validation must happen before any file access.
			OutputPath: filepath.Join(t.TempDir(), "does-not-exist"),
			Filename:   "whatever.mp4",
		}

		result, err := ConvertVideo(args, mediaType)
		if err == nil {
			t.Errorf("ConvertVideo(%q) returned no error, expected rejection", mediaType)
		}
		if result != nil {
			t.Errorf("ConvertVideo(%q) returned a non-nil result %+v", mediaType, result)
		}
	}
}

func TestConvertVideoAcceptsSupportedMediaTypes(t *testing.T) {
	// A supported type must get past validation and fail on the missing input
	// file instead, proving validation is not rejecting valid values.
	for _, mediaType := range []ConversionMediaType{Convert720p, Convert1080p} {
		args := &VideoConversionArgs{
			OutputPath: filepath.Join(t.TempDir(), "missing"),
			Filename:   "whatever.mp4",
		}

		_, err := ConvertVideo(args, mediaType)
		if err == nil {
			t.Fatalf("ConvertVideo(%q) unexpectedly succeeded", mediaType)
		}
		if got := err.Error(); !strings.Contains(got, "does not exist") {
			t.Errorf("ConvertVideo(%q) failed with %q, expected a missing-input error", mediaType, got)
		}
	}
}

// GetVideoInfo probes with -select_streams v:0, so an audio-only file yields an
// empty streams array. This used to panic with "index out of range [0] with
// length 0" and take the whole process down from a job worker.
func TestGetVideoInfoAudioOnlyReturnsError(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	src := filepath.Join("..", "..", "assets", "test.mp4")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("test asset missing: %v", err)
	}

	audioOnly := filepath.Join(t.TempDir(), "audio.mp3")
	cmd := exec.Command(ffmpeg, "-y", "-hide_banner", "-loglevel", "error",
		"-i", src, "-map", "a", "-q:a", "0", audioOnly)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("could not build audio-only fixture: %v: %s", err, out)
	}

	info, err := (&Video{FilePath: audioOnly}).GetVideoInfo()
	if err == nil {
		t.Fatalf("GetVideoInfo on an audio-only file returned no error: %+v", info)
	}
	if info != nil {
		t.Errorf("GetVideoInfo returned a non-nil info %+v alongside the error", info)
	}
}

// services.fixOrphanedFiles deletes a recording when CheckVideo fails, and
// distinguishes a genuine failure from a deliberate stop with
// errors.Is(err, ErrInterrupted). That only works while CheckVideo propagates
// ExecSync's error identity, so verify the real chain end to end: a wrapper
// using %v instead of %w here would silently start deleting healthy files.
func TestCheckVideoInterruptedIsIdentifiable(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available")
	}

	src := filepath.Join("..", "..", "assets", "test.mp4")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("test asset missing: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- CheckVideo(src) }()

	// Poll the registry for the ffmpeg process CheckVideo just started.
	var pid int
	for i := 0; i < 200 && pid == 0; i++ {
		processes.mu.Lock()
		for p := range processes.entries {
			pid = p
			break
		}
		processes.mu.Unlock()
		if pid == 0 {
			time.Sleep(5 * time.Millisecond)
		}
	}
	if pid == 0 {
		t.Skip("CheckVideo finished before it could be interrupted")
	}

	if err := Interrupt(pid); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}

	err := <-done
	if err == nil {
		t.Skip("CheckVideo completed before the signal landed")
	}
	if !errors.Is(err, ErrInterrupted) {
		t.Fatalf("CheckVideo returned %v, which does not satisfy errors.Is(err, ErrInterrupted); "+
			"fixOrphanedFiles would treat this as corruption and delete the recording", err)
	}
}

// Control: the normal video path still works.
func TestGetVideoInfoVideoFile(t *testing.T) {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not available")
	}

	src := filepath.Join("..", "..", "assets", "test.mp4")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("test asset missing: %v", err)
	}

	info, err := (&Video{FilePath: src}).GetVideoInfo()
	if err != nil {
		t.Fatalf("GetVideoInfo on a video file failed: %v", err)
	}
	if info.Width == 0 || info.Height == 0 || info.Duration <= 0 {
		t.Errorf("GetVideoInfo returned implausible metadata: %+v", info)
	}
}
