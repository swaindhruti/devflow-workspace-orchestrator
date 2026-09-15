//go:build windows

package runner

import (
	"os"
	"os/exec"
	"syscall"
)

// setProcessGroup is a no-op on Windows: os/exec has no equivalent of
// POSIX process groups there, so there's nothing to configure on cmd
// before starting it. See stopProcessGroup for the resulting
// limitation.
func setProcessGroup(cmd *exec.Cmd) {}

// stopProcessGroup can only signal the single process at pid on
// Windows — there's no process-group mechanism to fall back to here —
// so unlike on Unix, any child processes the command spawned may
// survive a Stop. Killing the whole tree would need a Windows Job
// Object, which isn't implemented yet.
func stopProcessGroup(pid int, _ syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Kill()
}
