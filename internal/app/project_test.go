package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	a, err := NewApp(testConfig(t), shellexec.NewFakeExecutor())
	if err != nil {
		t.Fatalf("failed to construct App: %v", err)
	}
	return a
}

func TestAddProjectDetectsDockerAutomatically(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if !p.HasDocker {
		t.Error("expected HasDocker = true")
	}
	if p.DockerComposeFile != "docker-compose.yml" {
		t.Errorf("DockerComposeFile = %q, want %q", p.DockerComposeFile, "docker-compose.yml")
	}
	if len(p.DockerCommands) != 5 {
		t.Errorf("expected 5 default docker commands, got %d", len(p.DockerCommands))
	}

	// Confirm it was actually persisted, not just returned.
	stored, err := a.Projects().GetProjectByID(p.ID)
	if err != nil {
		t.Fatalf("failed to reload project: %v", err)
	}
	if !stored.HasDocker {
		t.Error("expected persisted project to have HasDocker = true")
	}
}

func TestAddProjectWithoutDockerLeavesFieldsZero(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir() // empty: no Dockerfile, no compose file

	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if p.HasDocker {
		t.Error("expected HasDocker = false")
	}
	if len(p.DockerCommands) != 0 {
		t.Errorf("expected no docker commands, got %d", len(p.DockerCommands))
	}
}

func TestAddProjectPropagatesValidationError(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.AddProject("", "/some/path"); err == nil {
		t.Fatal("expected validation error for empty name, got nil")
	}
}

func TestRefreshDockerInfoUpdatesExistingProject(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir() // starts with no Docker setup
	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}
	if p.HasDocker {
		t.Fatal("expected project to start without Docker")
	}

	// Simulate the user adding a Dockerfile after registering the project.
	if err := os.WriteFile(filepath.Join(projectDir, "Dockerfile"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	updated, err := a.RefreshDockerInfo(p.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if !updated.HasDocker {
		t.Error("expected HasDocker = true after refresh")
	}
}

func TestRefreshDockerInfoReturnsErrorForMissingProject(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.RefreshDockerInfo("does-not-exist"); err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

func TestRefreshDockerInfoReturnsErrorForUnreadablePath(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir()
	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	// Simulate the project's directory having vanished since registration.
	if err := os.RemoveAll(projectDir); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	if _, err := a.RefreshDockerInfo(p.ID); err == nil {
		t.Fatal("expected error for unreadable path, got nil")
	}
}
