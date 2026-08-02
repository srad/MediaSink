package services

import (
	"sync"
	"testing"
)

// TryStart replaces a check-then-set pair that let two concurrent callers both
// begin a regeneration run.
func TestPreviewRegeneratorTryStartIsExclusive(t *testing.T) {
	pr := NewPreviewRegenerator()

	if !pr.TryStart() {
		t.Fatal("first TryStart returned false")
	}
	if pr.TryStart() {
		t.Fatal("second TryStart returned true while already running")
	}

	pr.Stop()

	if !pr.TryStart() {
		t.Fatal("TryStart returned false after Stop")
	}
	pr.Stop()
}

func TestPreviewRegeneratorTryStartConcurrent(t *testing.T) {
	pr := NewPreviewRegenerator()

	const goroutines = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	started := 0

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			if pr.TryStart() {
				mu.Lock()
				started++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if started != 1 {
		t.Fatalf("%d goroutines started a regeneration, want exactly 1", started)
	}
}

// Stop must reset the session so a failed run cannot wedge the flag on.
func TestPreviewRegeneratorStopResets(t *testing.T) {
	pr := NewPreviewRegenerator()

	if !pr.TryStart() {
		t.Fatal("TryStart returned false")
	}
	pr.SetTotal(10)
	pr.Update(3, "channel/video.mp4")

	pr.Stop()

	progress := pr.GetProgress()
	if progress.IsRunning || progress.Total != 0 || progress.Current != 0 || progress.CurrentVideo != "" {
		t.Fatalf("Stop did not reset the session: %+v", progress)
	}
}
