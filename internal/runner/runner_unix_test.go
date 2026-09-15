//go:build !windows

package runner

import (
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestStopKillsChildProcessesNotJustTheShell verifies that Stop tears
// down processes a command spawned as children of the shell, not just
// the shell itself. Before setProcessGroup/stopProcessGroup, Stop
// called cmd.Process.Kill(), which only signals the direct child Start
// invoked (the shell) — anything the shell itself forked (a
// backgrounded job here, standing in for what a real command like
// "npm run dev" or a docker-compose invocation commonly does
// internally) was left running, orphaned, since a killed parent
// doesn't take its children down with it.
func TestStopKillsChildProcessesNotJustTheShell(t *testing.T) {
	r := New(Config{})

	// Backgrounds a long sleep as a child of the shell, prints its PID
	// so the test can watch it directly, then waits on it — giving the
	// shell a child process of its own to survive (or not) a Stop.
	proc, err := r.Start("sleep 30 & echo $!; wait", "")
	if err != nil {
		t.Fatal(err)
	}

	var childPID int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if out := strings.TrimSpace(proc.Snapshot().Stdout); out != "" {
			if pid, err := strconv.Atoi(out); err == nil {
				childPID = pid
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if childPID == 0 {
		t.Fatal("did not observe the backgrounded child's PID in time")
	}

	if err := r.Stop(proc.ID); err != nil {
		t.Fatalf("failed to stop: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && proc.State() == StateRunning {
		time.Sleep(5 * time.Millisecond)
	}
	if state := proc.State(); state != StateStopped {
		t.Fatalf("expected stopped state, got %s", state)
	}

	// Signal 0 sends nothing — it's the standard way to probe whether a
	// process still exists (Kill returns an error, ESRCH, once it's
	// gone).
	deadline = time.Now().Add(2 * time.Second)
	var childErr error
	for time.Now().Before(deadline) {
		childErr = syscall.Kill(childPID, 0)
		if childErr != nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if childErr == nil {
		t.Fatalf("expected the backgrounded child (PID %d) to be gone after Stop, but it's still running", childPID)
	}
}
