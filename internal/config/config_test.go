package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Storage.Path == "" {
		t.Fatal("expected default storage path")
	}

	if cfg.Runner.Shell != "sh" {
		t.Fatalf("expected default shell 'sh', got '%s'", cfg.Runner.Shell)
	}

	if cfg.Runner.Timeout != "0" {
		t.Fatalf("expected default timeout '0', got '%s'", cfg.Runner.Timeout)
	}
}

func TestLoadConfig(t *testing.T) {
	content := []byte(`
storage:
  path: /tmp/test/projects.json
runner:
  shell: bash
  timeout: 30s
`)

	tmpFile, err := os.CreateTemp("", "devflow-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.Storage.Path != "/tmp/test/projects.json" {
		t.Fatalf("expected '/tmp/test/projects.json', got '%s'", cfg.Storage.Path)
	}

	if cfg.Runner.Shell != "bash" {
		t.Fatalf("expected 'bash', got '%s'", cfg.Runner.Shell)
	}

	if cfg.Runner.Timeout != "30s" {
		t.Fatalf("expected '30s', got '%s'", cfg.Runner.Timeout)
	}
}

func TestLoadConfigPartial(t *testing.T) {
	content := []byte(`
runner:
  shell: bash
`)

	tmpFile, err := os.CreateTemp("", "devflow-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Default storage path should be preserved
	expectedDefault := filepath.Join(os.Getenv("HOME"), ".devflow", "projects.json")
	if cfg.Storage.Path != expectedDefault {
		t.Fatalf("expected default path '%s', got '%s'", expectedDefault, cfg.Storage.Path)
	}

	// Overridden value should apply
	if cfg.Runner.Shell != "bash" {
		t.Fatalf("expected 'bash', got '%s'", cfg.Runner.Shell)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent/devflow.yaml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}

	if cfg.Storage.Path == "" {
		t.Fatal("expected defaults even when file not found")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	content := []byte(`invalid: yaml: :`)

	tmpFile, err := os.CreateTemp("", "devflow-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
