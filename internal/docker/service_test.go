package docker

import (
	"strings"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/runner"
)

func TestDefaultCommandsReturnsNilWhenNoDocker(t *testing.T) {
	cmds := DefaultCommands(DetectionResult{HasDocker: false}, "myapp")
	if cmds != nil {
		t.Errorf("expected nil commands, got %v", cmds)
	}
}

func TestDefaultCommandsComposeScoped(t *testing.T) {
	result := DetectionResult{HasDocker: true, ComposeFile: "docker-compose.yml", Identifier: "myapp"}

	cmds := DefaultCommands(result, "myapp")

	wantNames := []string{"Up", "Down", "Build", "Restart", "Logs"}
	if len(cmds) != len(wantNames) {
		t.Fatalf("expected %d commands, got %d", len(wantNames), len(cmds))
	}

	for i, name := range wantNames {
		if cmds[i].Name != name {
			t.Errorf("command %d name = %q, want %q", i, cmds[i].Name, name)
		}
		if !strings.Contains(cmds[i].Command, "docker compose -f docker-compose.yml") {
			t.Errorf("command %q does not reference the compose file: %q", name, cmds[i].Command)
		}
	}
}

func TestDefaultCommandsDockerfileOnly(t *testing.T) {
	result := DetectionResult{HasDocker: true}

	cmds := DefaultCommands(result, "myapp")

	wantNames := []string{"Build", "Run"}
	if len(cmds) != len(wantNames) {
		t.Fatalf("expected %d commands, got %d", len(wantNames), len(cmds))
	}

	for i, name := range wantNames {
		if cmds[i].Name != name {
			t.Errorf("command %d name = %q, want %q", i, cmds[i].Name, name)
		}
		if !strings.Contains(cmds[i].Command, "myapp") {
			t.Errorf("command %q does not reference the identifier: %q", name, cmds[i].Command)
		}
		if strings.Contains(cmds[i].Command, "compose") {
			t.Errorf("dockerfile-only command %q should not reference compose: %q", name, cmds[i].Command)
		}
	}
}

func TestDefaultCommandsGeneratesUniqueIDs(t *testing.T) {
	cmds := DefaultCommands(DetectionResult{HasDocker: true, ComposeFile: "docker-compose.yml"}, "myapp")

	seen := make(map[string]bool)
	for _, cmd := range cmds {
		if cmd.ID == "" {
			t.Fatal("expected every command to have a non-empty ID")
		}
		if seen[cmd.ID] {
			t.Fatalf("duplicate command ID generated: %s", cmd.ID)
		}
		seen[cmd.ID] = true
	}
}

func TestServiceExecuteFallsBackToProjectPath(t *testing.T) {
	svc := NewService(runner.New(runner.Config{}))

	cmd := DefaultCommands(DetectionResult{HasDocker: true}, "myapp")[0]
	cmd.Path = "" // force the fallback

	proc, err := svc.Execute(cmd, "/tmp")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if proc.WorkDir != "/tmp" {
		t.Errorf("expected process WorkDir %q, got %q", "/tmp", proc.WorkDir)
	}
}

func TestServiceExecutePrefersCommandPath(t *testing.T) {
	svc := NewService(runner.New(runner.Config{}))

	cmd := DefaultCommands(DetectionResult{HasDocker: true}, "myapp")[0]
	cmd.Path = "/var"

	proc, err := svc.Execute(cmd, "/tmp")
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}
	if proc.WorkDir != "/var" {
		t.Errorf("expected process WorkDir %q, got %q", "/var", proc.WorkDir)
	}
}
