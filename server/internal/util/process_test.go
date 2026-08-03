package util

import (
	"errors"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// The process registry used to be a bare map written by ExecSync and deleted
// from by Interrupt, which produced `fatal error: concurrent map writes` — an
// unrecoverable crash that gin.Recovery cannot catch. Run with -race.
func TestProcessRegistryConcurrentAccess(_ *testing.T) {
	var wg sync.WaitGroup

	// Stands in for the two job workers, both running ffmpeg via ExecSync.
	for worker := 0; worker < 6; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 40; i++ {
				_ = ExecSync(&ExecArgs{Command: "/bin/true"})
			}
		}()
	}

	// Stands in for POST /jobs/stop/:pid and the job.Pid interrupt paths.
	for caller := 0; caller < 4; caller++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for pid := 0; pid < 4000; pid++ {
				_ = Interrupt(pid)
			}
		}()
	}

	wg.Wait()
}

// Interrupt used to delete the map entry, which made ExecSync skip Wait() and
// return nil — reporting success for a process that had just been killed.
func TestExecSyncReportsInterruption(t *testing.T) {
	started := make(chan int, 1)
	done := make(chan error, 1)

	go func() {
		done <- ExecSync(&ExecArgs{
			Command:     "/bin/sleep",
			CommandArgs: []string{"30"},
			OnStart:     func(info CommandInfo) { started <- info.Pid },
		})
	}()

	var pid int
	select {
	case pid = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("process never started")
	}

	if err := Interrupt(pid); err != nil {
		t.Fatalf("Interrupt: %v", err)
	}

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("ExecSync returned nil for an interrupted process")
		}
		if !errors.Is(err, ErrInterrupted) {
			t.Fatalf("got %v, want an error wrapping ErrInterrupted", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ExecSync did not return after interruption")
	}
}

// Skipping Wait() also leaked the child as a zombie holding its pipe fds.
func TestInterruptedProcessIsReaped(t *testing.T) {
	started := make(chan int, 1)
	done := make(chan error, 1)

	go func() {
		done <- ExecSync(&ExecArgs{
			Command:     "/bin/sleep",
			CommandArgs: []string{"30"},
			OnStart:     func(info CommandInfo) { started <- info.Pid },
		})
	}()

	pid := <-started

	processes.mu.Lock()
	entry := processes.entries[pid]
	processes.mu.Unlock()
	if entry == nil {
		t.Fatal("process was not registered")
	}

	_ = Interrupt(pid)
	<-done

	if entry.cmd.ProcessState == nil {
		t.Fatal("ProcessState is nil: Wait() was skipped and the child leaked")
	}
}

// A process signalled so late that it still exits 0 completed its work. Callers
// such as the conversion handler delete their output on error, so misreporting
// this as an interruption would destroy a good result.
func TestInterruptedButSuccessfulExitReportsSuccess(t *testing.T) {
	errCh := make(chan error, 1)
	go func() {
		errCh <- ExecSync(&ExecArgs{
			Command: "/bin/true",
			OnStart: func(info CommandInfo) {
				// Mark it while ExecSync is still running, reproducing the race
				// between a normal exit and a stop request.
				_, _ = processes.markInterrupted(info.Pid)
			},
		})
	}()

	if err := <-errCh; err != nil {
		t.Fatalf("a successful process was reported as failed: %v", err)
	}
}

// A command that cannot start must return an error rather than panicking, and
// must not leave anything behind in the registry.
func TestExecSyncUnstartableCommand(t *testing.T) {
	err := ExecSync(&ExecArgs{Command: "/nonexistent/definitely-not-a-binary"})
	if err == nil {
		t.Fatal("expected an error for a command that cannot start")
	}
	if errors.Is(err, ErrInterrupted) {
		t.Fatalf("a start failure was misreported as interruption: %v", err)
	}

	processes.mu.Lock()
	remaining := len(processes.entries)
	processes.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("registry holds %d entries after a failed start", remaining)
	}
}

func TestExecSyncNormalExits(t *testing.T) {
	if err := ExecSync(&ExecArgs{Command: "/bin/true"}); err != nil {
		t.Fatalf("/bin/true returned %v, want nil", err)
	}

	err := ExecSync(&ExecArgs{Command: "/bin/false"})
	if err == nil {
		t.Fatal("/bin/false returned nil, want an error")
	}
	if errors.Is(err, ErrInterrupted) {
		t.Fatalf("a normal failure was misreported as interruption: %v", err)
	}
}

func TestInterruptUnknownPid(t *testing.T) {
	if err := Interrupt(999999); err != nil {
		t.Fatalf("Interrupt on an unknown pid returned %v, want nil", err)
	}
}

// Interrupt can race a normal exit and find a registered process that has
// already been reaped. os.ErrProcessDone means there is nothing left to stop,
// which is not a failure — and Job.updateStatus returns early on any error from
// here, skipping the status write, so reporting one would strand the job.
func TestInterruptAlreadyExitedProcess(t *testing.T) {
	c := exec.Command("/bin/true")
	if err := c.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := c.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}

	pid := c.Process.Pid
	processes.register(pid, c)
	defer processes.unregister(pid)

	if err := Interrupt(pid); err != nil {
		t.Fatalf("Interrupt on an already-exited process returned %v, want nil", err)
	}
}

// ExecSync's deferred unregister must always run, or the registry grows without
// bound as jobs are processed.
func TestRegistryIsDrainedAfterCompletion(t *testing.T) {
	if err := ExecSync(&ExecArgs{Command: "/bin/true"}); err != nil {
		t.Fatalf("ExecSync: %v", err)
	}

	processes.mu.Lock()
	remaining := len(processes.entries)
	processes.mu.Unlock()

	if remaining != 0 {
		t.Fatalf("registry still holds %d entries after completion", remaining)
	}
}
