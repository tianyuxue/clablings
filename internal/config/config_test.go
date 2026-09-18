package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFrom(filepath.Join(dir, "nope.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Containerlab.BinPath != "" || cfg.Docker.BinPath != "" {
		t.Fatalf("默认 bin_path 应为空: %+v", cfg)
	}
	if cfg.Build.DefaultImage != "gcc:13" {
		t.Fatalf("默认镜像 = %q", cfg.Build.DefaultImage)
	}
	if cfg.Watch.PollInterval != 250*time.Millisecond || cfg.Watch.Debounce != 300*time.Millisecond {
		t.Fatalf("默认监听参数 = %+v", cfg.Watch)
	}
}

func TestLoadFromOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("containerlab:\n  bin_path: /opt/clab\ndocker:\n  bin_path: /usr/bin/docker\nbuild:\n  default_image: gcc:12\nwatch:\n  poll_interval: 100ms\n  debounce: 50ms\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Containerlab.BinPath != "/opt/clab" || cfg.Build.DefaultImage != "gcc:12" {
		t.Fatalf("cfg = %+v", cfg)
	}
	if cfg.Watch.PollInterval != 100*time.Millisecond {
		t.Fatalf("poll = %v", cfg.Watch.PollInterval)
	}
}
