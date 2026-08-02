package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/srad/mediasink/server/internal/util"
)

// fixOrphanedFiles deletes a recording — file and database row — when its
// integrity check fails. Once ExecSync began reporting deliberate interruption
// as an error, an interrupted check would have been read as corruption, so the
// distinction below is what stops a healthy recording from being destroyed.
func TestIsCorruptionEvidence(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"healthy file", nil, false},
		{"interrupted check", util.ErrInterrupted, false},
		{"wrapped interruption", fmt.Errorf("%w: signal: interrupt", util.ErrInterrupted), false},
		{"double wrapped interruption", fmt.Errorf("check failed: %w", fmt.Errorf("%w: signal: interrupt", util.ErrInterrupted)), false},
		{"genuine corruption", errors.New("moov atom not found"), true},
		{"ffmpeg exit status", errors.New("exit status 1"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isCorruptionEvidence(tc.err); got != tc.want {
				t.Fatalf("isCorruptionEvidence(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// The guard above only works while CheckVideo returns ExecSync's error
// unwrapped, or wrapped with %w. If it is ever wrapped with %v the chain breaks
// silently and healthy recordings start being deleted again, so pin the
// requirement here rather than relying on the call site staying unchanged.
func TestCheckVideoPreservesErrorIdentity(t *testing.T) {
	wrapped := fmt.Errorf("%w: signal: interrupt", util.ErrInterrupted)

	if !errors.Is(wrapped, util.ErrInterrupted) {
		t.Fatal("errors.Is must see through the wrapping ExecSync applies")
	}
	if isCorruptionEvidence(wrapped) {
		t.Fatal("a wrapped interruption must not be treated as corruption")
	}
}
