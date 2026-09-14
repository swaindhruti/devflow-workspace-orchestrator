// Package git implements DevFlow's Git awareness domain: reading a
// project's branch, working tree status, ahead/behind tracking, and recent
// commit history. It reads this information by parsing the `git` CLI's
// machine-readable output (porcelain v2 for status, a tab-free oneline log
// for commits) rather than depending on a Git-implementation library like
// go-git, keeping with the project's "minimal external dependencies"
// principle. All CLI access goes through internal/shellexec.Executor
// rather than os/exec directly, so this package's parsing logic is unit
// testable without git actually being installed.
package git

// Status represents a project's Git working tree at a point in time, as
// reported by `git status --porcelain=v2 --branch`.
type Status struct {
	// Branch is the current branch name, or "" if the repository is in a
	// detached HEAD state (see Detached).
	Branch string
	// Detached is true if HEAD does not point at a branch tip.
	Detached bool
	// Clean is true if there are no staged changes, unstaged changes, or
	// untracked files — equivalent to Staged == 0 && Unstaged == 0 &&
	// Untracked == 0, precomputed here so callers don't need to repeat
	// that check.
	Clean bool
	// Staged is the number of tracked entries with a staged (index)
	// change.
	Staged int
	// Unstaged is the number of tracked entries with an unstaged
	// (working tree) change, including unmerged/conflicted entries.
	Unstaged int
	// Untracked is the number of files present in the working tree but
	// not tracked by Git and not ignored.
	Untracked int
	// Ahead is the number of commits the current branch has that its
	// upstream does not. Zero if there is no upstream configured.
	Ahead int
	// Behind is the number of commits the upstream has that the current
	// branch does not. Zero if there is no upstream configured.
	Behind int
}

// Commit represents a single entry from a project's commit history, as
// reported by `git log --oneline`.
type Commit struct {
	// Hash is the commit's abbreviated SHA, as Git itself renders it
	// (length depends on the repository's ambiguity settings).
	Hash string
	// Subject is the commit's first message line (its summary).
	Subject string
}
