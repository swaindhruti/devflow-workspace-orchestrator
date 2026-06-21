package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Storage StorageConfig `yaml:"storage"`
	Runner  RunnerConfig  `yaml:"runner"`
}

type StorageConfig struct {
	Path string `yaml:"path"`
}

type RunnerConfig struct {
	Shell   string `yaml:"shell"`
	Timeout string `yaml:"timeout"`
}

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
