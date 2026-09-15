package runner

import (
	"strings"
	"testing"
	"time"
)

func TestStartEcho(t *testing.T) {
	r := New(Config{})

	proc, err := r.Start("echo hello world", "")
	if err != nil {
		t.Fatal(err)
	}

	if proc.ID == "" {
		t.Fatal("expected process ID to be generated")
	}

	if proc.State() != StateRunning {
		t.Fatalf("expected running state, got %s", proc.State())
	}

	// Wait for completion
	time.Sleep(100 * time.Millisecond)

	snap := proc.Snapshot()

	if snap.State != StateCompleted {
		t.Fatalf("expected completed state, got %s", snap.State)
	}

	output := strings.TrimSpace(snap.Stdout)
	if output != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", output)
	}

	if snap.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", snap.ExitCode)
	}
}

func TestStartWithWorkDir(t *testing.T) {
	r := New(Config{})

	proc, err := r.Start("pwd", "/tmp")
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	snap := proc.Snapshot()
	output := strings.TrimSpace(snap.Stdout)

	if snap.State != StateCompleted {
		t.Fatalf("expected completed state, got %s", snap.State)
	}

	if output != "/tmp" {
		t.Fatalf("expected '/tmp', got '%s'", output)
	}
}

func TestStartInvalidCommand(t *testing.T) {
	r := New(Config{})

	_, err := r.Start("nonexistent_command_xyz", "")
	if err != nil {
		// Start may fail immediately if the command can't be launched
		// This is valid behavior — no process to track
		return
	}

	time.Sleep(200 * time.Millisecond)

	processes := r.List()
	if len(processes) == 0 {
		return
	}

	proc := processes[0]
	if proc.State() != StateFailed {
		t.Fatalf("expected failed state, got %s", proc.State())
	}
}

func TestStopProcess(t *testing.T) {
	r := New(Config{})

	proc, err := r.Start("sleep 30", "")
	if err != nil {
		t.Fatal(err)
	}

	if proc.State() != StateRunning {
		t.Fatalf("expected running state, got %s", proc.State())
	}

	err = r.Stop(proc.ID)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	if state := proc.State(); state != StateStopped {
		t.Fatalf("expected stopped state, got %s", state)
	}
}

func TestStopNonExistentProcess(t *testing.T) {
	r := New(Config{})

	err := r.Stop("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent process")
	}
}

func TestListProcesses(t *testing.T) {
	r := New(Config{})

	_, err := r.Start("echo a", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Start("echo b", "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Start("echo c", "")
	if err != nil {
		t.Fatal(err)
	}

	procs := r.List()
	if len(procs) != 3 {
		t.Fatalf("expected 3 processes, got %d", len(procs))
	}
}

func TestGetProcess(t *testing.T) {
	r := New(Config{})

	created, err := r.Start("echo test", "")
	if err != nil {
		t.Fatal(err)
	}

	found, err := r.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if found.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, found.ID)
	}
}

func TestGetNonExistentProcess(t *testing.T) {
	r := New(Config{})

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent process")
	}
}

// TestSnapshotIsRaceFreeWhileRunning drives Process.Snapshot from a tight
// concurrent loop while a process is still emitting output, so `go test
// -race` catches any unsynchronized access between the exec.Cmd output
// copiers / exit-watcher goroutine (both writers) and Snapshot (the
// reader) — this is exactly the access pattern a UI polling loop uses.
func TestSnapshotIsRaceFreeWhileRunning(t *testing.T) {
	r := New(Config{})

	proc, err := r.Start("echo a; sleep 0.02; echo b; sleep 0.02; echo c", "")
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			snap := proc.Snapshot()
			if snap.State != StateRunning {
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for process to finish")
	}

	snap := proc.Snapshot()
	if snap.State != StateCompleted {
		t.Fatalf("expected completed state, got %s", snap.State)
	}
	if got := strings.TrimSpace(snap.Stdout); got != "a\nb\nc" {
		t.Fatalf("expected 'a\\nb\\nc', got %q", got)
	}
}

func TestProcessIDUniqueness(t *testing.T) {
	r := New(Config{})

	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		proc, err := r.Start("echo unique", "")
		if err != nil {
			t.Fatal(err)
		}

		if ids[proc.ID] {
			t.Fatalf("duplicate ID generated: %s", proc.ID)
		}
		ids[proc.ID] = true
	}
}
