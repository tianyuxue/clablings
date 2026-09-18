// Package config 读取 clablings 用户配置。
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

// Config 是用户配置，字段缺省时使用默认值。
type Config struct {
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

// DefaultPath 返回 ~/.config/clablings/config.yaml。
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "clablings", "config.yaml")
}

// Load 从默认路径加载。
func Load() (*Config, error) { return LoadFrom(DefaultPath()) }

// LoadFrom 从指定路径加载；文件不存在返回默认配置。
func LoadFrom(path string) (*Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取配置 %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置 %s: %w", path, err)
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
