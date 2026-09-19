// Package config reads the clablings user configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPollInterval = 250 * time.Millisecond
	defaultDebounce     = 300 * time.Millisecond
	defaultBuildImage   = "gcc:13"
)

// Config is the user configuration; defaults are used for omitted fields.
type Config struct {
	Language     string `yaml:"language"`
	Containerlab struct {
		BinPath string `yaml:"bin_path"`
	} `yaml:"containerlab"`
	Docker struct {
		BinPath string `yaml:"bin_path"`
	} `yaml:"docker"`
	Watch struct {
		PollInterval time.Duration `yaml:"poll_interval"`
		Debounce     time.Duration `yaml:"debounce"`
	} `yaml:"watch"`
	Build struct {
		DefaultImage string `yaml:"default_image"`
	} `yaml:"build"`
}

// DefaultPath returns ~/.config/clablings/config.yaml.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "clablings", "config.yaml")
}

// Load loads from the default path.
func Load() (*Config, error) { return LoadFrom(DefaultPath()) }

// LoadFrom loads from the given path; returns the default configuration if the file is missing.
func LoadFrom(path string) (*Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		// Return a usable default configuration alongside the error so callers
		// that only warn can keep running safely.
		return defaults(), fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return defaults(), fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Watch.PollInterval <= 0 {
		cfg.Watch.PollInterval = defaultPollInterval
	}
	if cfg.Watch.Debounce <= 0 {
		cfg.Watch.Debounce = defaultDebounce
	}
	if cfg.Build.DefaultImage == "" {
		cfg.Build.DefaultImage = defaultBuildImage
	}
	return cfg, nil
}

func defaults() *Config {
	cfg := &Config{}
	cfg.Watch.PollInterval = defaultPollInterval
	cfg.Watch.Debounce = defaultDebounce
	cfg.Build.DefaultImage = defaultBuildImage
	return cfg
}
