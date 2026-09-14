package git

import (
	"errors"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

func TestStatusParsesCleanRepo(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte(
		"# branch.oid abc123\n# branch.head main\n",
	)}

	svc := NewService(exec)
	st, err := svc.Status("/proj")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	want := Status{Branch: "main", Clean: true}
	if st != want {
		t.Errorf("Status = %+v, want %+v", st, want)
	}
}

func TestStatusParsesDirtyRepoCounts(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte(
		"# branch.oid abc123\n" +
			"# branch.head main\n" +
			"1 M. N... 100644 100644 100644 aaa bbb staged_only.go\n" +
			"1 .M N... 100644 100644 100644 aaa bbb unstaged_only.go\n" +
			"1 MM N... 100644 100644 100644 aaa bbb both.go\n" +
			"u UU N... 100644 100644 100644 100644 aaa bbb ccc conflicted.go\n" +
			"? new_file.go\n" +
			"? another_new.go\n",
	)}

	svc := NewService(exec)
	st, err := svc.Status("/proj")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if st.Clean {
		t.Error("expected Clean = false for a dirty repo")
	}
	if st.Staged != 2 { // staged_only.go, both.go
		t.Errorf("Staged = %d, want 2", st.Staged)
	}
	if st.Unstaged != 3 { // unstaged_only.go, both.go, conflicted.go
		t.Errorf("Unstaged = %d, want 3", st.Unstaged)
	}
	if st.Untracked != 2 {
		t.Errorf("Untracked = %d, want 2", st.Untracked)
	}
}

func TestStatusParsesAheadBehind(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte(
		"# branch.oid abc123\n" +
			"# branch.head main\n" +
			"# branch.upstream origin/main\n" +
			"# branch.ab +2 -5\n",
	)}

	svc := NewService(exec)
	st, err := svc.Status("/proj")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if st.Ahead != 2 || st.Behind != 5 {
		t.Errorf("Ahead/Behind = %d/%d, want 2/5", st.Ahead, st.Behind)
	}
}

func TestStatusParsesDetachedHead(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte(
		"# branch.oid abc123\n# branch.head (detached)\n",
	)}

	svc := NewService(exec)
	st, err := svc.Status("/proj")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if !st.Detached {
		t.Error("expected Detached = true")
	}
	if st.Branch != "" {
		t.Errorf("expected empty Branch for detached HEAD, got %q", st.Branch)
	}
}

func TestStatusPassesPathAsWorkingDirectory(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte("# branch.head main\n")}

	svc := NewService(exec)
	if _, err := svc.Status("/proj"); err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(exec.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(exec.Calls))
	}
	if exec.Calls[0].Dir != "/proj" {
		t.Errorf("expected dir %q, got %q", "/proj", exec.Calls[0].Dir)
	}
}

func TestStatusWrapsExecutorError(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{
		Stderr: []byte("fatal: not a git repository"),
		Err:    errors.New("exit status 128"),
	}

	svc := NewService(exec)
	_, err := svc.Status("/proj")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRecentCommitsParsesLines(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte(
		"abc1234 fix(docker): handle malformed output\n" +
			"def5678 feat(docker): add inspector\n",
	)}

	svc := NewService(exec)
	commits, err := svc.RecentCommits("/proj", 2)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	want := Commit{Hash: "abc1234", Subject: "fix(docker): handle malformed output"}
	if commits[0] != want {
		t.Errorf("commits[0] = %+v, want %+v", commits[0], want)
	}
}

func TestRecentCommitsPassesCountArg(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte("")}

	svc := NewService(exec)
	if _, err := svc.RecentCommits("/proj", 5); err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(exec.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(exec.Calls))
	}
	if !containsArg(exec.Calls[0].Args, "-5") {
		t.Errorf("expected args to contain \"-5\", got %v", exec.Calls[0].Args)
	}
}

func TestRecentCommitsReturnsEmptySliceForEmptyOutput(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{Stdout: []byte("\n")}

	svc := NewService(exec)
	commits, err := svc.RecentCommits("/proj", 5)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected no commits, got %d", len(commits))
	}
}

func TestRecentCommitsWrapsExecutorError(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["git"] = shellexec.Response{
		Stderr: []byte("fatal: your current branch 'main' does not have any commits yet"),
		Err:    errors.New("exit status 128"),
	}

	svc := NewService(exec)
	_, err := svc.RecentCommits("/proj", 5)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// containsArg reports whether want is present anywhere in args.
func containsArg(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}
