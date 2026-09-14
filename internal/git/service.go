package git

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

// Service reads Git information for a project by invoking the `git` CLI
// through a shellexec.Executor. It holds no state of its own beyond that
// executor: every method takes the project path it should operate on, so
// one Service can be reused across every registered project.
type Service struct {
	exec shellexec.Executor
}

// NewService constructs a Service that runs git through the given
// Executor.
//
// Parameters:
//   - exec: the Executor used to invoke the git CLI. Pass a
//     shellexec.DefaultExecutor in production and a
//     shellexec.FakeExecutor in tests.
func NewService(exec shellexec.Executor) *Service {
	return &Service{exec: exec}
}

// Status reads the current branch, working tree cleanliness, and
// ahead/behind tracking for the repository at path.
//
// Parameters:
//   - path: the project directory to run git in. Must be inside a Git
//     working tree.
//
// Returns the parsed Status, or an error if git could not be invoked
// (e.g. path is not a Git repository) or produced output this method
// could not parse.
func (s *Service) Status(path string) (Status, error) {
	stdout, stderr, err := s.exec.Run(path, "git", "status", "--porcelain=v2", "--branch")
	if err != nil {
		return Status{}, fmt.Errorf("git status failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}

	st := parseStatus(string(stdout))
	return st, nil
}

// parseStatus interprets the output of `git status --porcelain=v2
// --branch`. It understands the branch header lines ("# branch.head",
// "# branch.ab") and, for the file-change lines, only the leading entry
// kind ("1"/"2" for ordinary changes, "u" for unmerged, "?" for
// untracked) and the two-character XY status code — it does not need the
// path or mode fields that follow, so it does not parse them.
func parseStatus(output string) Status {
	var st Status

	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			head := strings.TrimPrefix(line, "# branch.head ")
			if head == "(detached)" {
				st.Detached = true
			} else {
				st.Branch = head
			}

		case strings.HasPrefix(line, "# branch.ab "):
			st.Ahead, st.Behind = parseAheadBehind(strings.TrimPrefix(line, "# branch.ab "))

		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "):
			fields := strings.Fields(line)
			if len(fields) < 2 || len(fields[1]) != 2 {
				continue
			}
			xy := fields[1]
			if xy[0] != '.' {
				st.Staged++
			}
			if xy[1] != '.' {
				st.Unstaged++
			}

		case strings.HasPrefix(line, "u "):
			// Unmerged (conflicted) entries always need working-tree
			// resolution, so they are counted as unstaged rather than
			// trying to attribute them to one side of the conflict.
			st.Unstaged++

		case strings.HasPrefix(line, "? "):
			st.Untracked++
		}
	}

	st.Clean = st.Staged == 0 && st.Unstaged == 0 && st.Untracked == 0

	return st
}

// parseAheadBehind parses a "# branch.ab" line's value, formatted as
// "+<ahead> -<behind>" (e.g. "+2 -1"), returning (0, 0) if it is malformed
// rather than failing the whole Status parse over one unreadable line.
func parseAheadBehind(value string) (ahead, behind int) {
	fields := strings.Fields(value)
	if len(fields) != 2 {
		return 0, 0
	}

	a, errA := strconv.Atoi(strings.TrimPrefix(fields[0], "+"))
	b, errB := strconv.Atoi(strings.TrimPrefix(fields[1], "-"))
	if errA != nil || errB != nil {
		return 0, 0
	}

	return a, b
}

// RecentCommits reads the n most recent commits reachable from HEAD for
// the repository at path.
//
// Parameters:
//   - path: the project directory to run git in. Must be inside a Git
//     working tree.
//   - n: the maximum number of commits to return, most recent first.
//
// Returns the matching commits (fewer than n if the repository's history
// is shorter), or an error if git could not be invoked — including the
// case of a repository with no commits yet, which git itself reports as a
// failure.
func (s *Service) RecentCommits(path string, n int) ([]Commit, error) {
	stdout, stderr, err := s.exec.Run(path, "git", "log", fmt.Sprintf("-%d", n), "--oneline")
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w (%s)", err, strings.TrimSpace(string(stderr)))
	}

	var commits []Commit
	for _, line := range strings.Split(string(stdout), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		hash, subject, _ := strings.Cut(line, " ")
		commits = append(commits, Commit{Hash: hash, Subject: subject})
	}

	return commits, nil
}
