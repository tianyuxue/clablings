package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRootWalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ManifestName), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "chapters", "01-x", "exercises", "01-y")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := FindRoot(deep)
	if err != nil {
		t.Fatalf("FindRoot: %v", err)
	}
	if got != root {
		t.Fatalf("FindRoot = %q, want %q", got, root)
	}
}

func TestFindRootNotFound(t *testing.T) {
	if _, err := FindRoot(t.TempDir()); err == nil {
		t.Fatal("want error, got nil")
	}
}
