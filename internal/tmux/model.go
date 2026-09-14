// Package tmux implements DevFlow's tmux session-orchestration domain: it
// creates, lists, and kills tmux sessions on behalf of a project so a
// project's terminal layout can be started once and returned to later,
// rather than rebuilt by hand every time. Like internal/docker and
// internal/git, it invokes the real CLI (`tmux`) through
// internal/shellexec.Executor rather than a Go tmux library, so its
// command construction and output parsing are unit testable without tmux
// actually being installed — true of this development environment, which
// has neither docker nor tmux.
package tmux

// Session represents one tmux session, as reported by `tmux
// list-sessions`.
type Session struct {
	// Name is the session's name, as passed to `tmux new-session -s`.
	Name string
	// Windows is the number of windows currently open in the session.
	Windows int
	// Attached is true if at least one client is currently attached to
	// the session.
	Attached bool
	// Created is tmux's own session_created value: a Unix timestamp
	// (seconds since the epoch) rendered as a string. Kept as the raw
	// string tmux reports rather than parsed into a time.Time, since
	// callers that only need to display it (the eventual TUI) have no
	// need for a typed value, mirroring how internal/docker keeps
	// Container/Image timestamps as Docker's own raw strings.
	Created string
}
