package tmux

import (
	"errors"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

func TestCreateSessionInvokesCorrectCommand(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{}

	svc := NewService(exec)
	if err := svc.CreateSession("myapp", "/proj"); err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(exec.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(exec.Calls))
	}
	want := []string{"new-session", "-d", "-s", "myapp", "-c", "/proj"}
	if !argsEqual(exec.Calls[0].Args, want) {
		t.Errorf("args = %v, want %v", exec.Calls[0].Args, want)
	}
}

func TestCreateSessionWrapsError(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("duplicate session: myapp"),
		Err:    errors.New("exit status 1"),
	}

	svc := NewService(exec)
	if err := svc.CreateSession("myapp", "/proj"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListSessionsParsesRows(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{Stdout: []byte(
		"myapp\t3\t1\t1757840000\n" +
			"other\t1\t0\t1757830000\n",
	)}

	svc := NewService(exec)
	sessions, err := svc.ListSessions()
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	want := Session{Name: "myapp", Windows: 3, Attached: true, Created: "1757840000"}
	if sessions[0] != want {
		t.Errorf("sessions[0] = %+v, want %+v", sessions[0], want)
	}
	if sessions[1].Attached {
		t.Errorf("expected sessions[1].Attached = false")
	}
}

func TestListSessionsTreatsNoServerRunningAsEmpty(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("no server running on /tmp/tmux-1000/default"),
		Err:    errors.New("exit status 1"),
	}

	svc := NewService(exec)
	sessions, err := svc.ListSessions()
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected no sessions, got %d", len(sessions))
	}
}

func TestListSessionsWrapsOtherErrors(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("some other tmux failure"),
		Err:    errors.New("exit status 1"),
	}

	svc := NewService(exec)
	if _, err := svc.ListSessions(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestKillSessionInvokesCorrectCommand(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{}

	svc := NewService(exec)
	if err := svc.KillSession("myapp"); err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	want := []string{"kill-session", "-t", "myapp"}
	if !argsEqual(exec.Calls[0].Args, want) {
		t.Errorf("args = %v, want %v", exec.Calls[0].Args, want)
	}
}

func TestKillSessionWrapsError(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("can't find session: myapp"),
		Err:    errors.New("exit status 1"),
	}

	svc := NewService(exec)
	if err := svc.KillSession("myapp"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHasSessionReturnsTrueOnSuccess(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{}

	svc := NewService(exec)
	has, err := svc.HasSession("myapp")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if !has {
		t.Error("expected HasSession = true")
	}
}

func TestHasSessionReturnsFalseWhenNotFound(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("can't find session: myapp"),
		Err:    errors.New("exit status 1"),
	}

	svc := NewService(exec)
	has, err := svc.HasSession("myapp")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if has {
		t.Error("expected HasSession = false")
	}
}

func TestHasSessionWrapsOtherErrors(t *testing.T) {
	exec := shellexec.NewFakeExecutor()
	exec.Responses["tmux"] = shellexec.Response{
		Stderr: []byte("tmux: command not found"),
		Err:    errors.New("exit status 127"),
	}

	svc := NewService(exec)
	_, err := svc.HasSession("myapp")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAttachArgsReturnsExpectedCommand(t *testing.T) {
	name, args := AttachArgs("myapp")

	if name != "tmux" {
		t.Errorf("name = %q, want %q", name, "tmux")
	}
	want := []string{"attach-session", "-t", "myapp"}
	if !argsEqual(args, want) {
		t.Errorf("args = %v, want %v", args, want)
	}
}

func argsEqual(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
