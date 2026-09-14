package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

func TestProjectContextReturnsErrorForMissingProject(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.ProjectContext("does-not-exist"); err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

func TestProjectContextGitStatusNilForNonGitDirectory(t *testing.T) {
	// Uses a real shellexec.DefaultExecutor so this exercises the actual
	// git binary (available in this dev environment, unlike docker/tmux),
	// against a directory that deliberately isn't a Git repository.
	a, err := NewApp(testConfig(t), shellexec.DefaultExecutor{})
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}

	p, err := a.AddProject("myapp", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	ctx, err := a.ProjectContext(p.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if ctx.GitStatus != nil {
		t.Errorf("expected nil GitStatus for a non-Git directory, got %+v", ctx.GitStatus)
	}
}

func TestProjectContextIncludesGitStatusForRealGitRepo(t *testing.T) {
	repoDir := t.TempDir()
	if out, err := exec.Command("git", "init", repoDir).CombinedOutput(); err != nil {
		t.Fatalf("failed to set up git fixture repo: %v (%s)", err, out)
	}

	a, err := NewApp(testConfig(t), shellexec.DefaultExecutor{})
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}

	p, err := a.AddProject("myapp", repoDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	ctx, err := a.ProjectContext(p.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if ctx.GitStatus == nil {
		t.Fatal("expected non-nil GitStatus for a real git repo")
	}
	if !ctx.GitStatus.Clean {
		t.Errorf("expected a freshly initialized repo to be clean, got %+v", ctx.GitStatus)
	}
}

func TestProjectContextIncludesDockerContainersAndImages(t *testing.T) {
	fake := shellexec.NewFakeExecutor()
	fake.Responses["docker"] = shellexec.Response{
		Stdout: []byte(`{"ID":"abc123","Names":"myapp-web-1","Image":"myapp-web:latest","Status":"Up 2 hours","State":"running","Ports":"","CreatedAt":"now"}` + "\n"),
	}

	a, err := NewApp(testConfig(t), fake)
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	ctx, err := a.ProjectContext(p.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if len(ctx.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(ctx.Containers))
	}
	if ctx.Containers[0].Name != "myapp-web-1" {
		t.Errorf("container name = %q, want %q", ctx.Containers[0].Name, "myapp-web-1")
	}
}

func TestProjectContextDockerFieldsNilWhenInspectionFails(t *testing.T) {
	fake := shellexec.NewFakeExecutor()
	// No "docker" response registered: FakeExecutor.Run returns an error
	// for any call it wasn't told to handle, simulating docker being
	// unreachable (e.g. the daemon is down, or the binary is missing).

	a, err := NewApp(testConfig(t), fake)
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	ctx, err := a.ProjectContext(p.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if ctx.Containers != nil {
		t.Errorf("expected nil Containers when inspection fails, got %v", ctx.Containers)
	}
	if ctx.Images != nil {
		t.Errorf("expected nil Images when inspection fails, got %v", ctx.Images)
	}
}
