package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/config"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/state"
	"github.com/tianyuxue/clablings/internal/watch"
)

type fakeWatchEngine struct{ deployed, destroyed []string }

func (f *fakeWatchEngine) Deploy(_ context.Context, ch *content.Chapter) error {
	f.deployed = append(f.deployed, ch.ID)
	return nil
}
func (f *fakeWatchEngine) Destroy(_ context.Context, ch *content.Chapter) error {
	f.destroyed = append(f.destroyed, ch.ID)
	return nil
}

type fakeWatchReset struct{ fast []string }

func (f *fakeWatchReset) Fast(_ context.Context, ex *content.Exercise) error {
	f.fast = append(f.fast, ex.FullID())
	return nil
}
func (f *fakeWatchReset) Hard(_ context.Context, ex *content.Exercise) error { return nil }

func oneExerciseLibrary(t *testing.T) (*content.Library, *state.State) {
	t.Helper()
	ch := &content.Chapter{ID: "c1", TopologyName: "lab"}
	ex := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch}
	lib := &content.Library{
		Root:     t.TempDir(),
		Order:    []*content.Exercise{ex},
		ByID:     map[string]*content.Exercise{"c1/e1": ex},
		Chapters: []*content.Chapter{ch},
	}
	st, err := state.Load(lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	return lib, st
}

func TestVerifyAndReloadCompletes(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	next, done, err := verifyAndReload(context.Background(), sess, lib, cfg, &out, &out)
	if err != nil {
		t.Fatalf("verifyAndReload: %v", err)
	}
	if !done || next != nil {
		t.Fatalf("done=%v next=%v", done, next)
	}
	if !strings.Contains(out.String(), "all exercises complete") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestWatchLoopEnterCompletes(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	enterCh := make(chan struct{}, 1)
	enterCh <- struct{}{}
	tick := make(chan time.Time)
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, enterCh, tick, &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if !strings.Contains(out.String(), "all exercises complete") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestMakeVerifierFileGate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("// I AM NOT DONE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &content.Exercise{ID: "e1", Chapter: "c1", Dir: dir,
		Assertions: []content.Assertion{
			{Name: "file", Type: "file", Source: "main.c"},
			{Name: "exec", Type: "exec", Node: "r1", Cmd: "true"},
		}}
	eng := engine.New("containerlab", "docker")
	results, err := makeVerifier(eng)(context.Background(), ex)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if len(results) != 1 || results[0].Type != "file" || results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestMakeVerifierTopologyError(t *testing.T) {
	ch := &content.Chapter{ID: "c1", TopologyPath: filepath.Join(t.TempDir(), "missing.clab.yml")}
	ex := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch,
		Assertions: []content.Assertion{{Name: "x", Type: "file", Source: "missing"}}}
	eng := engine.New("containerlab", "docker")
	if _, err := makeVerifier(eng)(context.Background(), ex); err == nil {
		t.Fatal("expected topology parse error")
	}
}
