package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/project"
)

func TestRunCommandExecutesPlainCommand(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir()
	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	cmd := project.Command{ID: "cmd-1", Name: "Echo", Command: "echo hi"}
	p.AddCommand(cmd)
	if err := a.Projects().UpdateProject(p); err != nil {
		t.Fatalf("failed to save command: %v", err)
	}

	proc, err := a.RunCommand(p.ID, "cmd-1")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if proc.Command != "echo hi" {
		t.Errorf("Command = %q, want %q", proc.Command, "echo hi")
	}
	if proc.WorkDir != projectDir {
		t.Errorf("WorkDir = %q, want %q", proc.WorkDir, projectDir)
	}
}

func TestRunCommandExecutesDockerCommand(t *testing.T) {
	a := newTestApp(t)

	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yml"), []byte{}, 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	p, err := a.AddProject("myapp", projectDir)
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}
	if len(p.DockerCommands) == 0 {
		t.Fatal("expected default docker commands to be populated")
	}

	upCmd := p.DockerCommands[0]

	proc, err := a.RunCommand(p.ID, upCmd.ID)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if proc.Command != upCmd.Command {
		t.Errorf("Command = %q, want %q", proc.Command, upCmd.Command)
	}
	if proc.WorkDir != projectDir {
		t.Errorf("WorkDir = %q, want %q (docker.Service.Execute should fall back to the project path)", proc.WorkDir, projectDir)
	}
}

func TestRunCommandReturnsErrorForMissingProject(t *testing.T) {
	a := newTestApp(t)

	if _, err := a.RunCommand("does-not-exist", "cmd-1"); err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

func TestRunCommandReturnsErrorForUnknownCommandID(t *testing.T) {
	a := newTestApp(t)

	p, err := a.AddProject("myapp", t.TempDir())
	if err != nil {
		t.Fatalf("failed to add project: %v", err)
	}

	if _, err := a.RunCommand(p.ID, "does-not-exist"); err == nil {
		t.Fatal("expected error for unknown command ID, got nil")
	}
}
