// Package config 读取 clablings 用户配置。
package config

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
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
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.Watch.PollInterval <= 0 {
		cfg.Watch.PollInterval = 250 * time.Millisecond
	}
	if cfg.Watch.Debounce <= 0 {
		cfg.Watch.Debounce = 300 * time.Millisecond
	}
	if cfg.Build.DefaultImage == "" {
		cfg.Build.DefaultImage = "gcc:13"
	}
	return cfg, nil
}

func defaults() *Config {
	cfg := &Config{}
	cfg.Watch.PollInterval = 250 * time.Millisecond
	cfg.Watch.Debounce = 300 * time.Millisecond
	cfg.Build.DefaultImage = "gcc:13"
	return cfg
}
