package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tianyuxue/clablings/internal/config"
)

func TestConfigOverrideDrivesBins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("containerlab:\n  bin_path: /opt/clab\ndocker:\n  bin_path: /opt/docker\nbuild:\n  default_image: gcc:12\nwatch:\n  poll_interval: 100ms\n  debounce: 40ms\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got := binOr(cfg.Containerlab.BinPath, "containerlab"); got != "/opt/clab" {
		t.Fatalf("clab bin = %q", got)
	}
	if got := binOr(cfg.Docker.BinPath, "docker"); got != "/opt/docker" {
		t.Fatalf("docker bin = %q", got)
	}
	if cfg.Build.DefaultImage != "gcc:12" {
		t.Fatalf("image = %q", cfg.Build.DefaultImage)
	}
	if cfg.Watch.PollInterval.String() != "100ms" || cfg.Watch.Debounce.String() != "40ms" {
		t.Fatalf("watch = %+v", cfg.Watch)
	}
}

func TestConfigFallbackWhenMissing(t *testing.T) {
	cfg, err := config.LoadFrom(filepath.Join(t.TempDir(), "absent.yaml"))
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if got := binOr(cfg.Containerlab.BinPath, "containerlab"); got != "containerlab" {
		t.Fatalf("fallback clab = %q", got)
	}
	if got := binOr(cfg.Docker.BinPath, "docker"); got != "docker" {
		t.Fatalf("fallback docker = %q", got)
	}
	if cfg.Build.DefaultImage != "gcc:13" {
		t.Fatalf("default image = %q", cfg.Build.DefaultImage)
	}
}
