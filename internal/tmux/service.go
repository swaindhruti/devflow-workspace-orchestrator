package tmux

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

// sessionListFormat is the `tmux list-sessions -F` format string used by
// ListSessions. Fields are tab-separated (tmux session names cannot
// contain tabs) in the same order Session's fields are populated, so
// parsing is a single strings.Split per line.
const sessionListFormat = "#{session_name}\t#{session_windows}\t#{session_attached}\t#{session_created}"

// Service orchestrates tmux sessions by invoking the `tmux` CLI through a
// shellexec.Executor. It holds no state beyond that executor, the same
// shape as internal/git.Service.
type Service struct {
	exec shellexec.Executor
}

// NewService constructs a Service that runs tmux through the given
// Executor.
//
// Parameters:
//   - exec: the Executor used to invoke the tmux CLI. Pass a
//     shellexec.DefaultExecutor in production and a
//     shellexec.FakeExecutor in tests.
func NewService(exec shellexec.Executor) *Service {
	return &Service{exec: exec}
}

// CreateSession starts a new detached tmux session.
//
// Parameters:
//   - name: the session name to create. Typically derived from the
//     project, the same way project.DockerIdentifier scopes Docker
//     queries.
//   - workDir: the directory the session's initial window starts in
//     (usually the project's own Path).
//
// Returns an error if tmux could not be invoked, including the case where
// a session named name already exists.
func (s *Service) CreateSession(name, workDir string) error {
	_, stderr, err := s.exec.Run("", "tmux", "new-session", "-d", "-s", name, "-c", workDir)
	if err != nil {
		return fmt.Errorf("tmux new-session failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}
	return nil
}

// ListSessions returns every tmux session currently known to the tmux
// server.
//
// Returns the parsed sessions, or an error if tmux could not be invoked
// for a reason other than "no server running". No tmux server having been
// started yet is not treated as an error — it is indistinguishable in
// intent from "zero sessions exist", so ListSessions returns an empty, nil
// slice for that case rather than forcing every caller to special-case it,
// mirroring how internal/project's JSONFileStorage treats a missing
// registry file as an empty project list rather than a failure.
func (s *Service) ListSessions() ([]Session, error) {
	stdout, stderr, err := s.exec.Run("", "tmux", "list-sessions", "-F", sessionListFormat)
	if err != nil {
		if isNoServerRunning(stderr) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}

	var sessions []Session
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			continue
		}

		windows, _ := strconv.Atoi(fields[1])
		attachedCount, _ := strconv.Atoi(fields[2])

		sessions = append(sessions, Session{
			Name:     fields[0],
			Windows:  windows,
			Attached: attachedCount > 0,
			Created:  fields[3],
		})
	}

	return sessions, nil
}

// KillSession terminates a tmux session.
//
// Parameters:
//   - name: the session name to terminate.
//
// Returns an error if tmux could not be invoked, including the case where
// no session named name exists.
func (s *Service) KillSession(name string) error {
	_, stderr, err := s.exec.Run("", "tmux", "kill-session", "-t", name)
	if err != nil {
		return fmt.Errorf("tmux kill-session failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}
	return nil
}

// HasSession reports whether a tmux session with the given name currently
// exists.
//
// Parameters:
//   - name: the session name to check for.
//
// Returns (true, nil) if the session exists, (false, nil) if it does not
// (tmux's normal "no such session" outcome, not treated as an error, the
// same way GetProjectByID's "not found" is a return value rather than a
// panic), or (false, err) if tmux could not be invoked at all (e.g. the
// tmux binary itself is missing).
func (s *Service) HasSession(name string) (bool, error) {
	_, stderr, err := s.exec.Run("", "tmux", "has-session", "-t", name)
	if err == nil {
		return true, nil
	}
	if isSessionNotFound(stderr) {
		return false, nil
	}
	return false, fmt.Errorf("tmux has-session failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
}

// AttachArgs returns the literal command and arguments used to attach to a
// tmux session: ("tmux", ["attach-session", "-t", name]).
//
// Parameters:
//   - name: the session name to attach to.
//
// This is deliberately a pure function, not a Service method that runs the
// command through shellexec.Executor. Attaching to a tmux session hands
// the real terminal (stdin/stdout/stderr, raw mode) over to the tmux
// client for the life of the session; every other call in this package
// (and in internal/docker, internal/git) runs a command to completion and
// reads back its captured output, which is a fundamentally different
// shape. Executing this is left to the future application/UI layer, which
// will need to run it with a real *exec.Cmd wired directly to the
// terminal's stdin/stdout/stderr — likely after suspending the Bubble Tea
// program — rather than through the capture-based Executor interface.
func AttachArgs(name string) (string, []string) {
	return "tmux", []string{"attach-session", "-t", name}
}

// isNoServerRunning reports whether stderr matches tmux's message for "no
// tmux server has been started yet" (e.g. when listing sessions before any
// have been created).
func isNoServerRunning(stderr []byte) bool {
	return strings.Contains(strings.ToLower(string(stderr)), "no server running")
}

// isSessionNotFound reports whether stderr matches tmux's message for "no
// session with this name exists".
func isSessionNotFound(stderr []byte) bool {
	return strings.Contains(strings.ToLower(string(stderr)), "can't find session")
}
