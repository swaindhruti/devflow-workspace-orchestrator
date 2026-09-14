// Package shellexec provides a small abstraction over running external CLI
// commands and capturing their output. Domain packages that need to query
// a CLI tool synchronously for a parsed result (docker, git, tmux) depend
// on the Executor interface rather than calling os/exec directly, so that
// logic can be unit tested with a FakeExecutor regardless of whether the
// real binary (docker, git, tmux) is installed on the machine running the
// tests.
//
// This is distinct from internal/runner, which tracks long-running,
// asynchronous, streamed processes such as `npm run dev`. shellexec is for
// short-lived calls whose output is read once, after the command has
// already finished, such as `git status --porcelain` or
// `docker ps --format json`.
package shellexec

import (
	"bytes"
	"os/exec"
)

// Executor runs an external command and returns its captured output. It is
// the seam that lets docker, git, and tmux domain logic be tested without
// those binaries actually being installed.
type Executor interface {
	// Run executes name with the given args and returns its captured
	// standard output and standard error.
	//
	// Parameters:
	//   - dir: the working directory to run the command from, or "" to
	//     use the calling process's own working directory.
	//   - name: the executable to run (e.g. "docker", "git", "tmux").
	//   - args: the arguments to pass to name.
	//
	// Returns the command's captured stdout and stderr as byte slices.
	// err is non-nil if the command could not be started or exited with a
	// non-zero status; stderr is still populated in that case so callers
	// can surface the underlying tool's own error message.
	Run(dir, name string, args ...string) (stdout, stderr []byte, err error)
}

// DefaultExecutor is the production Executor implementation, backed by
// os/exec. Its zero value is ready to use.
type DefaultExecutor struct{}

// Run implements Executor by invoking name as a real OS process via
// os/exec.Command, capturing its stdout and stderr into separate buffers
// rather than combining them, so callers can distinguish command output
// from diagnostic/error output.
func (DefaultExecutor) Run(dir, name string, args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err = cmd.Run()

	return outBuf.Bytes(), errBuf.Bytes(), err
}
