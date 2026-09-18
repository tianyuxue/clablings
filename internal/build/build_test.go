package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

type fakeRunner struct {
	calls   [][]string
	results []engine.Result
	errs    []error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (engine.Result, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	i := len(f.calls) - 1
	var res engine.Result
	var err error
	if i < len(f.results) {
		res = f.results[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return res, err
}

func TestBuildRunsDockerWithDeclaredImage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "arp_responder"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{}
	b := New(fr, "docker", "gcc:13")

	ex := &content.Exercise{
		ID: "01", Dir: dir,
		Build: &content.BuildSpec{Image: "gcc:13", Command: "make", Output: "arp_responder"},
	}
	bin, err := b.Build(context.Background(), ex)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if bin != filepath.Join(dir, "arp_responder") {
		t.Fatalf("bin = %q", bin)
	}
	want := []string{
		"docker", "run", "--rm",
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		"-v", dir + ":/src", "-w", "/src",
		"gcc:13", "sh", "-lc", "make",
	}
	if !reflect.DeepEqual(fr.calls[0], want) {
		t.Fatalf("calls[0] = %v, want %v", fr.calls[0], want)
	}
}

func TestBuildFallsBackToDefaultImage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "out"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRunner{}
	b := New(fr, "docker", "gcc:13")
	ex := &content.Exercise{ID: "01", Dir: dir, Build: &content.BuildSpec{Command: "make", Output: "out"}}
	if _, err := b.Build(context.Background(), ex); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if fr.calls[0][9] != "gcc:13" {
		t.Fatalf("image = %q, want default gcc:13", fr.calls[0][9])
	}
}

func TestBuildCompileFailureReturnsOutput(t *testing.T) {
	dir := t.TempDir()
	fr := &fakeRunner{results: []engine.Result{{Stderr: "error: expected ';'", Code: 1}}}
	b := New(fr, "docker", "gcc:13")
	ex := &content.Exercise{ID: "01", Dir: dir, Build: &content.BuildSpec{Command: "make", Output: "out"}}
	_, err := b.Build(context.Background(), ex)
	if err == nil {
		t.Fatal("expected a build failure error")
	}
	if got := err.Error(); !strings.Contains(got, "expected ';'") {
		t.Fatalf("error missing compiler output: %v", err)
	}
}
