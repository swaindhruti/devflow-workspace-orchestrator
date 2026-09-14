// Package config loads DevFlow's runtime configuration from a YAML file on
// disk, falling back to sane defaults for any value the file does not
// provide (or when the file does not exist at all). It is a shared
// infrastructure package: every domain package (project, docker, git, tmux)
// and the runner package read their tunables from the Config values loaded
// here, rather than reading environment variables or files themselves.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure for DevFlow. It groups the
// settings for each subsystem that needs to be configurable, currently
// project storage and the shell command runner.
type Config struct {
	// Storage controls where and how the project registry is persisted.
	Storage StorageConfig `yaml:"storage"`
	// Runner controls how shell commands are executed on behalf of a
	// project (which shell binary to invoke and how long to allow a
	// command to run before it is considered timed out).
	Runner RunnerConfig `yaml:"runner"`
}

// StorageConfig holds settings for the project registry's persistence
// layer.
type StorageConfig struct {
	// Path is the filesystem location of the JSON file that stores all
	// registered projects. It is created (along with any missing parent
	// directories) on first write if it does not already exist.
	Path string `yaml:"path"`
}

// RunnerConfig holds settings for the process runner used to execute
// project and Docker commands.
type RunnerConfig struct {
	// Shell is the shell binary used to interpret command strings, e.g.
	// "sh" or "bash". It is invoked as `<shell> -c "<command>"`.
	Shell string `yaml:"shell"`
	// Timeout is the maximum duration a command may run before it is
	// forcibly stopped, expressed as a string (e.g. "30s", "5m"). A value
	// of "0" means no timeout is enforced.
	Timeout string `yaml:"timeout"`
}

// Default returns a Config populated with DevFlow's built-in defaults: a
// project registry stored under "~/.devflow/projects.json", commands run
// through "sh", and no timeout. Default is used both as the starting point
// for Load (so a partial or missing config file only overrides the fields
// it actually sets) and by any caller that wants to run DevFlow without a
// config file at all.
func Default() *Config {
	return &Config{
		Storage: StorageConfig{
			Path: filepath.Join(os.Getenv("HOME"), ".devflow", "projects.json"),
		},
		Runner: RunnerConfig{
			Shell:   "sh",
			Timeout: "0",
		},
	}
}

// Load reads and parses the YAML configuration file at path, returning a
// Config with any values present in the file overlaid on top of Default's
// built-in defaults.
//
// Parameters:
//   - path: filesystem path to a YAML config file (e.g. "devflow.yaml").
//
// Returns the parsed Config, or an error if the file exists but could not
// be read or contains invalid YAML. A missing file at path is not treated
// as an error: Load silently falls back to Default() in that case, since
// DevFlow is expected to run with zero required configuration.
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
