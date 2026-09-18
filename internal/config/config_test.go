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
		t.Fatalf("default bin_path should be empty: %+v", cfg)
	}
	if cfg.Build.DefaultImage != "gcc:13" {
		t.Fatalf("default image = %q", cfg.Build.DefaultImage)
	}
	if cfg.Watch.PollInterval != 250*time.Millisecond || cfg.Watch.Debounce != 300*time.Millisecond {
		t.Fatalf("default watch params = %+v", cfg.Watch)
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
	if cfg.Docker.BinPath != "/usr/bin/docker" {
		t.Fatalf("docker = %q", cfg.Docker.BinPath)
	}
	if cfg.Watch.PollInterval != 100*time.Millisecond {
		t.Fatalf("poll = %v", cfg.Watch.PollInterval)
	}
	if cfg.Watch.Debounce != 50*time.Millisecond {
		t.Fatalf("debounce = %v", cfg.Watch.Debounce)
	}
}

func TestLoadFromFallbackNormalization(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantPoll     time.Duration
		wantDebounce time.Duration
		wantImage    string
	}{
		{
			name:         "zero durations and empty image",
			yaml:         "watch:\n  poll_interval: 0s\n  debounce: 0s\nbuild:\n  default_image: \"\"\n",
			wantPoll:     250 * time.Millisecond,
			wantDebounce: 300 * time.Millisecond,
			wantImage:    "gcc:13",
		},
		{
			name:         "negatives normalized",
			yaml:         "watch:\n  poll_interval: -1s\n  debounce: -1s\n",
			wantPoll:     250 * time.Millisecond,
			wantDebounce: 300 * time.Millisecond,
			wantImage:    "gcc:13",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadFrom(path)
			if err != nil {
				t.Fatalf("LoadFrom: %v", err)
			}
			if cfg.Watch.PollInterval != tt.wantPoll {
				t.Fatalf("poll = %v, want %v", cfg.Watch.PollInterval, tt.wantPoll)
			}
			if cfg.Watch.Debounce != tt.wantDebounce {
				t.Fatalf("debounce = %v, want %v", cfg.Watch.Debounce, tt.wantDebounce)
			}
			if cfg.Build.DefaultImage != tt.wantImage {
				t.Fatalf("image = %q, want %q", cfg.Build.DefaultImage, tt.wantImage)
			}
		})
	}
}

func TestLoadFromInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("build: [unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err == nil {
		t.Fatal("invalid YAML should return an error")
	}
	if cfg == nil || cfg.Watch.PollInterval != defaultPollInterval || cfg.Build.DefaultImage != defaultBuildImage {
		t.Fatalf("invalid YAML should still return a usable default config, got %+v", cfg)
	}
}

func TestLoadFromReadErrorReturnsDefaults(t *testing.T) {
	dir := t.TempDir() // a directory cannot be read as a file
	cfg, err := LoadFrom(dir)
	if err == nil {
		t.Fatal("reading a directory should return an error")
	}
	if cfg == nil || cfg.Watch.Debounce != defaultDebounce {
		t.Fatalf("read error should still return defaults, got %+v", cfg)
	}
}

func TestLoadFromPartialOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("containerlab:\n  bin_path: /x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Containerlab.BinPath != "/x" {
		t.Fatalf("containerlab = %q", cfg.Containerlab.BinPath)
	}
	if cfg.Docker.BinPath != "" {
		t.Fatalf("docker = %q", cfg.Docker.BinPath)
	}
	if cfg.Watch.Debounce != 300*time.Millisecond {
		t.Fatalf("debounce = %v", cfg.Watch.Debounce)
	}
	if cfg.Build.DefaultImage != "gcc:13" {
		t.Fatalf("image = %q", cfg.Build.DefaultImage)
	}
}
