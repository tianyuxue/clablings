//go:build integration

package build

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestIntegrationBuildInContainer(t *testing.T) {
	dir := t.TempDir()
	src := "#include <stdio.h>\nint main(void){ printf(\"hello clablings\\n\"); return 0; }\n"
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mk := "main: main.c\n\tcc -static -O2 -o main main.c\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(mk), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New("containerlab", "docker")
	b := New(eng.Runner(), "docker", "gcc:13")
	ex := &content.Exercise{ID: "01", Dir: dir, Build: &content.BuildSpec{Command: "make", Output: "main"}}
	bin, err := b.Build(context.Background(), ex)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	out, err := exec.Command(bin).Output()
	if err != nil {
		t.Fatalf("run artifact: %v", err)
	}
	if string(out) != "hello clablings\n" {
		t.Fatalf("output = %q", out)
	}
}
