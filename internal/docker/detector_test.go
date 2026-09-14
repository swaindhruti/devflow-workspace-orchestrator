package docker

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFiles creates each named empty file inside dir, failing the test on
// any error, so table-driven tests can set up a project directory's
// contents in one line.
func writeFiles(t *testing.T, dir string, names ...string) {
	t.Helper()

	for _, name := range names {
		if err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644); err != nil {
			t.Fatalf("failed to write fixture file %q: %v", name, err)
		}
	}
}

func TestDetectDockerSetup(t *testing.T) {
	tests := []struct {
		name            string
		files           []string
		wantHasDocker   bool
		wantComposeFile string
		wantIdentifier  bool // whether an Identifier should be derived at all
	}{
		{
			name:          "no docker files present",
			files:         []string{"main.go", "README.md"},
			wantHasDocker: false,
		},
		{
			name:          "dockerfile only, no compose",
			files:         []string{"Dockerfile"},
			wantHasDocker: true,
		},
		{
			name:            "docker-compose.yml",
			files:           []string{"docker-compose.yml"},
			wantHasDocker:   true,
			wantComposeFile: "docker-compose.yml",
			wantIdentifier:  true,
		},
		{
			name:            "docker-compose.yaml",
			files:           []string{"docker-compose.yaml"},
			wantHasDocker:   true,
			wantComposeFile: "docker-compose.yaml",
			wantIdentifier:  true,
		},
		{
			name:            "compose.yml (v2 naming)",
			files:           []string{"compose.yml"},
			wantHasDocker:   true,
			wantComposeFile: "compose.yml",
			wantIdentifier:  true,
		},
		{
			name:            "dockerfile and compose file both present prefers compose",
			files:           []string{"Dockerfile", "docker-compose.yml"},
			wantHasDocker:   true,
			wantComposeFile: "docker-compose.yml",
			wantIdentifier:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFiles(t, dir, tt.files...)

			result, err := DetectDockerSetup(dir)
			if err != nil {
				t.Fatalf("did not expect error but got %v", err)
			}

			if result.HasDocker != tt.wantHasDocker {
				t.Errorf("HasDocker = %v, want %v", result.HasDocker, tt.wantHasDocker)
			}
			if result.ComposeFile != tt.wantComposeFile {
				t.Errorf("ComposeFile = %q, want %q", result.ComposeFile, tt.wantComposeFile)
			}
			if tt.wantIdentifier && result.Identifier == "" {
				t.Errorf("expected a non-empty Identifier, got empty")
			}
			if !tt.wantIdentifier && result.Identifier != "" {
				t.Errorf("expected no Identifier, got %q", result.Identifier)
			}
		})
	}
}

func TestDetectDockerSetupIdentifierMatchesComposeNormalization(t *testing.T) {
	parent := t.TempDir()
	projectDir := filepath.Join(parent, "My Cool App!")
	if err := os.Mkdir(projectDir, 0755); err != nil {
		t.Fatalf("failed to create project dir: %v", err)
	}
	writeFiles(t, projectDir, "docker-compose.yml")

	result, err := DetectDockerSetup(projectDir)
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	want := "mycoolapp"
	if result.Identifier != want {
		t.Errorf("Identifier = %q, want %q", result.Identifier, want)
	}
}

func TestDetectDockerSetupReturnsErrorForMissingDirectory(t *testing.T) {
	_, err := DetectDockerSetup(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected error for missing directory, got nil")
	}
}
