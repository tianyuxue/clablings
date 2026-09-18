package cli

import (
	"bytes"
	"context"
	"errors"
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

func TestWatchLoopVerifyErrorDoesNotPanic(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return nil, errors.New("boom")
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	enterCh := make(chan struct{}, 1)
	tick := make(chan time.Time, 2)
	enterCh <- struct{}{}

	var out, errOut bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- watchLoop(ctx, sess, lib, cfg, enterCh, tick, &out, &errOut) }()

	time.Sleep(20 * time.Millisecond) // let enter be processed
	tick <- time.Now()                // previously panicked on the nil watcher
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watchLoop: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("watchLoop did not return after cancel")
	}
	if !strings.Contains(errOut.String(), "verify error") {
		t.Fatalf("errOut = %q", errOut.String())
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
	results, err := makeVerifier(eng, nil)(context.Background(), ex)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if len(results) != 1 || results[0].Type != "file" || results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
}

type fakeRebuilder struct {
	calls int
	err   error
}

func (f *fakeRebuilder) Rebuild(context.Context, *content.Exercise) error {
	f.calls++
	return f.err
}

type stubRunner struct{ calls [][]string }

func (s *stubRunner) Run(_ context.Context, name string, args ...string) (engine.Result, error) {
	s.calls = append(s.calls, append([]string{name}, args...))
	return engine.Result{}, nil
}

func TestMakeVerifierBuildStep(t *testing.T) {
	ex := &content.Exercise{ID: "e1", Chapter: "c1", Type: content.TypeCodeImplement,
		Assertions: []content.Assertion{{Name: "exec", Type: "exec", Node: "r1", Cmd: "true"}}}
	eng := engine.NewWithRunner(&stubRunner{}, "containerlab", "docker")

	// success: rebuild runs, then the exec assertion passes
	rb := &fakeRebuilder{}
	results, err := makeVerifier(eng, rb)(context.Background(), ex)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if rb.calls != 1 || len(results) != 1 || !results[0].Passed {
		t.Fatalf("calls=%d results=%+v", rb.calls, results)
	}

	// build failure: a failed build result is returned and the rest is skipped
	rb = &fakeRebuilder{err: errors.New("compile boom")}
	results, err = makeVerifier(eng, rb)(context.Background(), ex)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if len(results) != 1 || results[0].Passed || results[0].Type != "build" {
		t.Fatalf("results = %+v", results)
	}
}

func TestMakeVerifierBuildRequiresRebuilder(t *testing.T) {
	ex := &content.Exercise{ID: "e1", Chapter: "c1", Type: content.TypeCodeImplement}
	results, err := makeVerifier(engine.New("containerlab", "docker"), nil)(context.Background(), ex)
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	if len(results) != 1 || results[0].Passed || results[0].Actual != "no rebuilder configured" {
		t.Fatalf("results = %+v", results)
	}
}

func TestMakeVerifierTopologyError(t *testing.T) {
	ch := &content.Chapter{ID: "c1", TopologyPath: filepath.Join(t.TempDir(), "missing.clab.yml")}
	ex := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch,
		Assertions: []content.Assertion{{Name: "x", Type: "file", Source: "missing"}}}
	eng := engine.New("containerlab", "docker")
	if _, err := makeVerifier(eng, nil)(context.Background(), ex); err == nil {
		t.Fatal("expected topology parse error")
	}
}
