package runner

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type State int

const (
	StateRunning State = iota
	StateCompleted
	StateFailed
	StateStopped
)

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

type Process struct {
	ID         string
	Command    string
	WorkDir    string
	State      State
	PID        int
	cmd        *exec.Cmd
	Stdout     bytes.Buffer
	Stderr     bytes.Buffer
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int
	stopped    bool
}

type Config struct {
	Shell   string
	Timeout string
}

type Runner struct {
	cfg       Config
	mu        sync.Mutex
	processes map[string]*Process
}

func New(cfg Config) *Runner {
	if cfg.Shell == "" {
		cfg.Shell = "sh"
	}

	return &Runner{
		cfg:       cfg,
		processes: make(map[string]*Process),
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

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

func (r *Runner) Get(id string) (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	proc, ok := r.processes[id]
	if !ok {
		return nil, fmt.Errorf("process not found: %s", id)
	}

	return proc, nil
}

func (r *Runner) List() []*Process {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]*Process, 0, len(r.processes))
	for _, proc := range r.processes {
		result = append(result, proc)
	}

	return result
}
