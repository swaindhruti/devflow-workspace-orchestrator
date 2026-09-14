package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/config"
	"github.com/swaindhruti/devflow-workspace-orchestrator.git/internal/shellexec"
)

// testConfig builds a Config pointing at a fresh temp directory's
// projects.json, so each test gets an isolated, disposable registry file.
func testConfig(t *testing.T) *config.Config {
	t.Helper()
	dir := t.TempDir()
	return &config.Config{
		Storage: config.StorageConfig{Path: filepath.Join(dir, "projects.json")},
		Runner:  config.RunnerConfig{Shell: "sh", Timeout: "0"},
	}
}

func TestNewAppWiresAllServices(t *testing.T) {
	cfg := testConfig(t)

	a, err := NewApp(cfg, shellexec.NewFakeExecutor())
	if err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if a.Projects() == nil {
		t.Error("expected Projects() to be non-nil")
	}
	if a.Docker() == nil {
		t.Error("expected Docker() to be non-nil")
	}
	if a.Git() == nil {
		t.Error("expected Git() to be non-nil")
	}
	if a.Tmux() == nil {
		t.Error("expected Tmux() to be non-nil")
	}
}

func TestNewAppCreatesStorageDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{Path: filepath.Join(dir, "nested", "registry", "projects.json")},
		Runner:  config.RunnerConfig{Shell: "sh"},
	}

	if _, err := NewApp(cfg, shellexec.NewFakeExecutor()); err != nil {
		t.Fatalf("did not expect error but got %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "nested", "registry")); err != nil {
		t.Errorf("expected storage directory to be created, stat failed: %v", err)
	}
}

func TestNewAppReturnsErrorWhenStorageDirCannotBeCreated(t *testing.T) {
	dir := t.TempDir()
	blockingFile := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(blockingFile, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to set up fixture: %v", err)
	}

	cfg := &config.Config{
		Storage: config.StorageConfig{Path: filepath.Join(blockingFile, "sub", "projects.json")},
		Runner:  config.RunnerConfig{Shell: "sh"},
	}

	if _, err := NewApp(cfg, shellexec.NewFakeExecutor()); err == nil {
		t.Fatal("expected error when storage directory cannot be created, got nil")
	}
}
