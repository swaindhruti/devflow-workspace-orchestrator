//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

// setProcessGroup configures cmd to run as the leader of its own new
// process group (Setpgid with Pgid left at 0, which makes the child's
// own PID the new group ID), so stopProcessGroup can later signal the
// whole group at once instead of just the one process Start returns a
// handle to.
//
// This matters because every command here runs through a shell
// (`<Shell> -c "<command>"`): things like "npm run dev", a
// docker-compose invocation, or a "build && run" pair commonly have the
// shell fork additional child processes rather than exec'ing directly
// into a single one. Killing only the shell's own PID — what
// cmd.Process.Kill() does — leaves those children running and orphaned,
// since a killed parent doesn't take its children down with it.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopProcessGroup sends sig to every process in pid's process group —
// the negative-PID convention kill(2) uses for "signal the whole
// group" — rather than just pid itself, so a command's child processes
// are stopped along with it. pid must belong to a process started with
// setProcessGroup; otherwise the negated PID may not correspond to any
// process group at all.
func stopProcessGroup(pid int, sig syscall.Signal) error {
	return syscall.Kill(-pid, sig)
}
