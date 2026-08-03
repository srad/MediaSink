package onnx

import (
	"sync"
	"testing"
)

// The library path used to be read straight from ONNXRUNTIME_LIB inside
// EnsureInitialized's sync.Once. It is now handed in by the composition root, so
// what EnsureInitialized consumes must be what was set.
func TestSetLibraryPath(t *testing.T) {
	original := sharedLibraryPath()
	t.Cleanup(func() { SetLibraryPath(original) })

	SetLibraryPath("/opt/onnx/libonnxruntime.so")
	if got := sharedLibraryPath(); got != "/opt/onnx/libonnxruntime.so" {
		t.Errorf("sharedLibraryPath() = %q, want the value just set", got)
	}

	// Empty means "use the system default", not "keep the previous value".
	SetLibraryPath("")
	if got := sharedLibraryPath(); got != "" {
		t.Errorf("sharedLibraryPath() = %q, want empty", got)
	}
}

// The setter runs at startup while EnsureInitialized may already be reading from
// another goroutine, so the pair has to be synchronised. Run with -race.
func TestSetLibraryPathIsRaceFree(t *testing.T) {
	original := sharedLibraryPath()
	t.Cleanup(func() { SetLibraryPath(original) })

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); SetLibraryPath("/some/path") }()
		go func() { defer wg.Done(); _ = sharedLibraryPath() }()
	}
	wg.Wait()
}
