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
	"syscall"
	"time"
)

// Snapshot is a consistent, point-in-time copy of a Process's mutable
// state, taken under a single lock so State, Stdout, Stderr, ExitCode, and
// FinishedAt never appear inconsistent relative to one another (e.g. State
// already StateCompleted but Stdout missing output written just before
// exit). Callers that need to observe a running Process — a UI polling
// loop, in particular — should always go through Process.Snapshot rather
// than reading multiple fields separately.
type Snapshot struct {
	State      State
	Stdout     string
	Stderr     string
	ExitCode   int
	FinishedAt time.Time
}

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
//
// ID, Command, WorkDir, PID, and StartedAt are set once, before the
// Process is ever handed to another goroutine, so they're safe to read
// directly with no lock. Everything that can change while the process
// runs — state, captured output, exit info — is guarded by mu and must be
// read through State or Snapshot instead of accessed as a field: a
// background goroutine (started in Start) writes to it for as long as the
// process is alive, so an unguarded read of those fields while
// State() == StateRunning is a data race.
type Process struct {
	// ID uniquely identifies this process within the Runner that started
	// it.
	ID string
	// Command is the raw shell command string that was executed.
	Command string
	// WorkDir is the working directory the command was run from, or ""
	// if it ran in the Runner's own working directory.
	WorkDir string
	// PID is the operating-system process ID assigned once the command
	// has started.
	PID int
	// StartedAt is the time the process was started.
	StartedAt time.Time

	// cmd is the underlying exec.Cmd, kept so Stop can signal the
	// running process.
	cmd *exec.Cmd

	// mu guards every field below, all of which can be mutated for as
	// long as the process is running.
	mu sync.Mutex
	// state is the process's current lifecycle state. It starts at
	// StateRunning and transitions exactly once, when the process exits,
	// to StateCompleted, StateFailed, or StateStopped.
	state State
	// stdout accumulates everything the process has written to standard
	// output since it started.
	stdout bytes.Buffer
	// stderr accumulates everything the process has written to standard
	// error since it started.
	stderr bytes.Buffer
	// finishedAt is the time the process exited. It is the zero Time
	// while state is StateRunning.
	finishedAt time.Time
	// exitCode is the process's OS exit code once it has finished. It is
	// meaningless while state is StateRunning.
	exitCode int
	// stopped records whether Stop was called for this process, so the
	// exit-handling goroutine can distinguish "killed by us" from other
	// abnormal exits when computing the final state.
	stopped bool
}

// State returns the process's current lifecycle state. Safe to call at
// any time, including while the process is still running.
func (p *Process) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Snapshot returns a consistent, point-in-time copy of the process's
// state and captured output so far. Safe to call at any time, including
// repeatedly from a polling loop while the process is still running —
// this is the intended way for a caller (e.g. a UI) to observe a
// Process's progress.
func (p *Process) Snapshot() Snapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	return Snapshot{
		State:      p.state,
		Stdout:     p.stdout.String(),
		Stderr:     p.stderr.String(),
		ExitCode:   p.exitCode,
		FinishedAt: p.finishedAt,
	}
}

// processWriter adapts a Process's mu-guarded output buffer to io.Writer,
// so exec.Cmd's own stdout/stderr copying goroutines write through the
// same lock Snapshot reads through, rather than writing directly into a
// bytes.Buffer with no synchronization at all.
type processWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w processWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
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
		state:     StateRunning,
		cmd:       cmd,
		StartedAt: time.Now(),
	}

	cmd.Stdout = processWriter{mu: &proc.mu, buf: &proc.stdout}
	cmd.Stderr = processWriter{mu: &proc.mu, buf: &proc.stderr}
	setProcessGroup(cmd)

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

		proc.mu.Lock()
		defer proc.mu.Unlock()

		proc.finishedAt = time.Now()
		if err != nil {
			if proc.stopped {
				proc.state = StateStopped
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				proc.exitCode = exitErr.ExitCode()
				proc.state = StateFailed
			} else {
				proc.state = StateStopped
			}
		} else {
			proc.exitCode = 0
			proc.state = StateCompleted
		}
	}()

	return proc, nil
}

// Stop forcibly kills a running process — on Unix, its entire process
// group (see setProcessGroup/stopProcessGroup), not just the top-level
// shell Start invoked. Killing only that shell (what a plain
// cmd.Process.Kill() does) can leave its children running: a command
// like "npm run dev" or a docker-compose invocation commonly has the
// shell fork additional processes rather than exec'ing directly into
// one, and a killed parent doesn't take orphaned children down with it
// — worse, if a surviving child still holds the shell's inherited
// stdout/stderr pipe open, cmd.Wait() in Start's exit-watcher goroutine
// never even returns, since Cmd.Wait also waits for those pipes to see
// EOF, leaving State stuck at StateRunning indefinitely.
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

	proc.mu.Lock()
	if proc.state != StateRunning {
		proc.mu.Unlock()
		return fmt.Errorf("process %s is not running", id)
	}
	proc.stopped = true
	proc.mu.Unlock()

	return stopProcessGroup(proc.PID, syscall.SIGKILL)
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
