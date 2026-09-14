// Package runner executes arbitrary shell commands as tracked, asynchronous
// processes. It is shared infrastructure rather than a domain package: it
// knows nothing about projects, Docker, or Git — it just starts a command
// through a configured shell, records its state, and captures its
// stdout/stderr, so any domain package (project commands today; Docker
// compose actions from Phase 1 onward) can run a command without
// reimplementing process bookkeeping.
package runner

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// State represents where a Process is in its lifecycle.
type State int

const (
	// StateRunning means the process has started and has not yet exited.
	StateRunning State = iota
	// StateCompleted means the process exited with status code 0.
	StateCompleted
	// StateFailed means the process exited with a non-zero status code.
	StateFailed
	// StateStopped means the process was killed via Runner.Stop, or
	// exited abnormally for a reason other than a non-zero exit code.
	StateStopped
)

// String returns the lowercase, human-readable name of the state (e.g.
// "running"), used when displaying process status in the UI or logs.
func (s State) String() string {
	switch s {
	case StateRunning:
		return "running"
	case StateCompleted:
		return "completed"
	case StateFailed:
		return "failed"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// Process represents a single shell command that has been started by a
// Runner, along with its captured output and current lifecycle state.
type Process struct {
	// ID uniquely identifies this process within the Runner that started
	// it.
	ID string
	// Command is the raw shell command string that was executed.
	Command string
	// WorkDir is the working directory the command was run from, or ""
	// if it ran in the Runner's own working directory.
	WorkDir string
	// State is the process's current lifecycle state. It starts at
	// StateRunning and transitions exactly once, when the process exits,
	// to StateCompleted, StateFailed, or StateStopped.
	State State
	// PID is the operating-system process ID assigned once the command
	// has started.
	PID int
	// cmd is the underlying exec.Cmd, kept so Stop can signal the
	// running process.
	cmd *exec.Cmd
	// Stdout accumulates everything the process has written to standard
	// output since it started.
	Stdout bytes.Buffer
	// Stderr accumulates everything the process has written to standard
	// error since it started.
	Stderr bytes.Buffer
	// StartedAt is the time the process was started.
	StartedAt time.Time
	// FinishedAt is the time the process exited. It is the zero Time
	// while State is StateRunning.
	FinishedAt time.Time
	// ExitCode is the process's OS exit code once it has finished. It is
	// meaningless while State is StateRunning.
	ExitCode int
	// stopped records whether Stop was called for this process, so the
	// exit-handling goroutine can distinguish "killed by us" from other
	// abnormal exits when computing the final State.
	stopped bool
}

// Config holds the settings a Runner uses to execute every command it
// starts.
type Config struct {
	// Shell is the shell binary used to interpret command strings (e.g.
	// "sh", "bash"). Commands are executed as `<Shell> -c "<command>"`.
	// If empty, New defaults it to "sh".
	Shell string
	// Timeout is currently accepted for forward compatibility with a
	// per-command execution timeout but is not yet enforced by Start.
	Timeout string
}

// Runner starts and tracks shell commands as Process values. It is safe for
// concurrent use: all access to the internal process map is guarded by mu.
type Runner struct {
	cfg       Config
	mu        sync.Mutex
	processes map[string]*Process
}

// New constructs a Runner with the given Config. If cfg.Shell is empty, it
// defaults to "sh" so a zero-value Config still produces a working Runner.
//
// Parameters:
//   - cfg: the shell and timeout settings to use for every command this
//     Runner starts.
func New(cfg Config) *Runner {
	if cfg.Shell == "" {
		cfg.Shell = "sh"
	}

	return &Runner{
		cfg:       cfg,
		processes: make(map[string]*Process),
	}
}

// generateID produces a random 32-character hex string suitable for use as
// a Process.ID.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// Start launches command through the Runner's configured shell and begins
// tracking it as a Process. The command runs asynchronously: Start returns
// as soon as the process has been spawned, not when it finishes. A
// background goroutine watches for the process to exit and updates its
// State, FinishedAt, and ExitCode accordingly.
//
// Parameters:
//   - command: the shell command line to execute (e.g. "go test ./...").
//   - workDir: the working directory to run the command from, or "" to use
//     the Runner process's own working directory.
//
// Returns the new Process (already registered with the Runner and
// retrievable via Get/List), or an error if the command could not be
// started (e.g. the shell binary was not found).
func (r *Runner) Start(command, workDir string) (*Process, error) {
	cmd := exec.Command(r.cfg.Shell, "-c", command)
	if workDir != "" {
		cmd.Dir = workDir
	}

	proc := &Process{
		ID:        generateID(),
		Command:   command,
		WorkDir:   workDir,
		State:     StateRunning,
		cmd:       cmd,
		StartedAt: time.Now(),
	}

	cmd.Stdout = &proc.Stdout
	cmd.Stderr = &proc.Stderr

	err := cmd.Start()
	if err != nil {
		return nil, err
	}

	proc.PID = cmd.Process.Pid

	r.mu.Lock()
	r.processes[proc.ID] = proc
	r.mu.Unlock()

	go func() {
		err := cmd.Wait()
		r.mu.Lock()
		defer r.mu.Unlock()

		proc.FinishedAt = time.Now()
		if err != nil {
			if proc.stopped {
				proc.State = StateStopped
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				proc.ExitCode = exitErr.ExitCode()
				proc.State = StateFailed
			} else {
				proc.State = StateStopped
			}
		} else {
			proc.ExitCode = 0
			proc.State = StateCompleted
		}
	}()

	return proc, nil
}

// Stop forcibly kills a running process.
//
// Parameters:
//   - id: the Process.ID to stop, as returned by Start.
//
// Returns an error if no process with that ID is tracked, if the process
// is not currently in StateRunning, or if sending the kill signal failed.
// On success, the process's State transitions to StateStopped once its
// exit is observed by the goroutine started in Start.
func (r *Runner) Stop(id string) error {
	r.mu.Lock()
	proc, ok := r.processes[id]
	r.mu.Unlock()

	if !ok {
		return fmt.Errorf("process not found: %s", id)
	}

	if proc.State != StateRunning {
		return fmt.Errorf("process %s is not running", id)
	}

	proc.stopped = true
	return proc.cmd.Process.Kill()
}

// Get returns the tracked Process matching id.
//
// Parameters:
//   - id: the Process.ID to look up, as returned by Start.
//
// Returns an error if no process with that ID is tracked by this Runner.
func (r *Runner) Get(id string) (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proc, ok := r.processes[id]
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return proc, nil
}

// List returns every process this Runner has started, in no particular
// order, including ones that have already finished.
func (r *Runner) List() []*Process {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]*Process, 0, len(r.processes))
	for _, proc := range r.processes {
		result = append(result, proc)
	}

	return result
}
