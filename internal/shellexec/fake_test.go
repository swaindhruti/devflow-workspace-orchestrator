package shellexec

import (
	"errors"
	"testing"
)

func TestFakeExecutorRecordsCallsAndReturnsRegisteredResponse(t *testing.T) {
	f := NewFakeExecutor()
	f.Responses["docker"] = Response{Stdout: []byte("container-1\n")}

	stdout, _, err := f.Run("/proj", "docker", "ps", "-a")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if string(stdout) != "container-1\n" {
		t.Fatalf("expected registered stdout, got %q", stdout)
	}

	if len(f.Calls) != 1 {
		t.Fatalf("expected 1 recorded call, got %d", len(f.Calls))
	}

	got := f.Calls[0]
	if got.Dir != "/proj" {
		t.Errorf("expected recorded dir %q, got %q", "/proj", got.Dir)
	}
	if got.Name != "docker" {
		t.Errorf("expected recorded name %q, got %q", "docker", got.Name)
	}
	if len(got.Args) != 2 || got.Args[0] != "ps" || got.Args[1] != "-a" {
		t.Errorf("expected recorded args [ps -a], got %v", got.Args)
	}
}

func TestFakeExecutorReturnsRegisteredError(t *testing.T) {
	f := NewFakeExecutor()
	wantErr := errors.New("docker: command not found")
	f.Responses["docker"] = Response{Err: wantErr}

	_, _, err := f.Run("", "docker", "ps")
	if err != wantErr {
		t.Fatalf("expected registered error %v, got %v", wantErr, err)
	}
}

func TestFakeExecutorErrorsOnUnregisteredCommand(t *testing.T) {
	f := NewFakeExecutor()

	_, _, err := f.Run("", "docker", "ps")
	if err == nil {
		t.Fatal("expected error for unregistered command, got nil")
	}
}
