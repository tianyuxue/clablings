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
	ch := &content.Chapter{ID: "c1", TopologyPath: "c1/topology.clab.yml", TopologyName: "lab"}
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
	done, err := verifyAndReload(context.Background(), sess, lib, cfg, &out, &out)
	if err != nil {
		t.Fatalf("verifyAndReload: %v", err)
	}
	if !done {
		t.Fatalf("done=%v", done)
	}
	if !strings.Contains(out.String(), "all exercises complete") {
		t.Fatalf("out = %q", out.String())
	}
	if !strings.Contains(out.String(), "verifying") {
		t.Fatalf("out = %q, want a verifying status", out.String())
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
	keyCh := make(chan byte, 1)
	keyCh <- '\n'
	tick := make(chan time.Time)
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &out); err != nil {
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
	keyCh := make(chan byte, 1)
	tick := make(chan time.Time, 2)
	keyCh <- '\n'

	var out, errOut bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- watchLoop(ctx, sess, lib, cfg, "en", keyCh, tick, &out, &errOut) }()

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

func TestWatchLoopHintAndCleanQuit(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	eng := &fakeWatchEngine{}
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: eng, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	sess.Current().Hints = []string{"check the MTU"}
	keyCh := make(chan byte, 2)
	keyCh <- 'h'
	keyCh <- 'x'
	tick := make(chan time.Time)
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &out) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watchLoop: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchLoop did not return")
	}
	if !strings.Contains(out.String(), "check the MTU") {
		t.Fatalf("out = %q", out.String())
	}
	if len(eng.destroyed) != 1 || eng.destroyed[0] != "c1" {
		t.Fatalf("destroyed = %v, want [c1]", eng.destroyed)
	}
}

func TestWatchLoopQuitNotice(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	keyCh := make(chan byte, 1)
	keyCh <- 'q'
	tick := make(chan time.Time)
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if !strings.Contains(out.String(), "containers kept running") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestWatchLoopConceptKey(t *testing.T) {
	dir := t.TempDir()
	chDir := filepath.Join(dir, "chapters", "c1")
	if err := os.MkdirAll(chDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(chDir, "concept.en.md"), []byte("# Switching\n\nwhy L2.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lib, st := oneExerciseLibrary(t)
	lib.Chapters[0].Dir = chDir
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	keyCh := make(chan byte, 2)
	keyCh <- 'c'
	keyCh <- 'q'
	tick := make(chan time.Time)
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if !strings.Contains(out.String(), "why L2.") {
		t.Fatalf("concept not shown:\n%s", out.String())
	}
}

func TestVerifyAndReloadQuizAborted(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	lib.Order[0].Type = content.TypeQuiz
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Quiz: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return nil, watch.ErrQuizAborted
		},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	done, err := verifyAndReload(context.Background(), sess, lib, cfg, &out, &out)
	if err != nil {
		t.Fatalf("verifyAndReload: %v", err)
	}
	if !done {
		t.Fatalf("done=%v", done)
	}
	if !strings.Contains(out.String(), "quiz aborted") {
		t.Fatalf("out = %q, want aborted notice", out.String())
	}
	if strings.Contains(out.String(), "verifying") {
		t.Fatalf("out = %q, quiz must not print a verifying status", out.String())
	}
	if !strings.Contains(out.String(), "containers kept running") {
		t.Fatalf("out = %q, want container disposition notice", out.String())
	}
}

func TestWatchLoopQuizStaysOnFailure(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	lib.Order[0].Type = content.TypeQuiz
	cfg, _ := config.LoadFrom("")
	called := 0
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Quiz: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			called++
			return []assert.Result{{Passed: false}}, nil
		},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
		Report: func([]assert.Result) {},
	})
	ctx, cancel := context.WithCancel(context.Background())
	if err := sess.Start(ctx); err != nil {
		t.Fatal(err)
	}
	keyCh := make(chan byte, 1)
	keyCh <- '\n'
	tick := make(chan time.Time)
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- watchLoop(ctx, sess, lib, cfg, "en", keyCh, tick, &out, &out) }()
	time.Sleep(30 * time.Millisecond)
	if called != 1 {
		t.Fatalf("quiz handler called %d times", called)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watchLoop: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("watchLoop did not return after cancel")
	}
}

func TestVerifyAndReloadQuizCanceledQuits(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	lib.Order[0].Type = content.TypeQuiz
	cfg, _ := config.LoadFrom("")
	ctx, cancel := context.WithCancel(context.Background())
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Quiz: func(qctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
			cancel()
			return nil, qctx.Err()
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	var out bytes.Buffer
	done, err := verifyAndReload(ctx, sess, lib, cfg, &out, &out)
	if err != nil || !done {
		t.Fatalf("done=%v err=%v out=%q", done, err, out.String())
	}
	if !strings.Contains(out.String(), "quiz aborted") {
		t.Fatalf("out = %q, want aborted notice", out.String())
	}
	if !strings.Contains(out.String(), "containers kept running") {
		t.Fatalf("out = %q, want container disposition notice", out.String())
	}
}

func TestWatchLoopEnterStartsQuiz(t *testing.T) {
	ch := &content.Chapter{ID: "c1"}
	ex := &content.Exercise{ID: "q1", Chapter: "c1", Type: content.TypeQuiz, ChapterRef: ch}
	ch.Exercises = []*content.Exercise{ex}
	lib := &content.Library{Root: t.TempDir(), Chapters: []*content.Chapter{ch},
		Order: []*content.Exercise{ex}, ByID: map[string]*content.Exercise{"c1/q1": ex}}
	st, _ := state.Load(lib.Root)
	cfg, _ := config.LoadFrom("")
	called := 0
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Quiz: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			called++
			return []assert.Result{{Passed: true}}, nil
		},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	keyCh := make(chan byte, 1)
	keyCh <- '\n'
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, make(chan time.Time), &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if called != 1 {
		t.Fatalf("quiz handler called %d times", called)
	}
}

func TestWatchLoopDrainsTypeAheadAfterQuiz(t *testing.T) {
	ch := &content.Chapter{ID: "c1"}
	q := &content.Exercise{ID: "q1", Chapter: "c1", Type: content.TypeQuiz, ChapterRef: ch}
	e2 := &content.Exercise{ID: "e2", Chapter: "c1", Type: content.TypeConfigFix, ChapterRef: ch,
		Hints: []string{"UNIQUE_HINT_MARKER"}}
	ch.Exercises = []*content.Exercise{q, e2}
	lib := &content.Library{Root: t.TempDir(), Chapters: []*content.Chapter{ch},
		Order: []*content.Exercise{q, e2}, ByID: map[string]*content.Exercise{"c1/q1": q, "c1/e2": e2}}
	st, _ := state.Load(lib.Root)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Quiz: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	byteCh := make(chan byte, 3)
	byteCh <- '\n'
	byteCh <- 'h'
	byteCh <- 'q'
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", byteCh, make(chan time.Time), &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if strings.Contains(out.String(), "UNIQUE_HINT_MARKER") {
		t.Fatalf("type-ahead was not drained:\n%s", out.String())
	}
}

func TestVerifyAndReloadNonQuizCancelIsNotAbort(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return nil, context.Canceled
		},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	done, err := verifyAndReload(context.Background(), sess, lib, cfg, &out, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if done {
		t.Fatal("non-quiz cancellation must not be treated as a completed quiz")
	}
	if strings.Contains(out.String(), "quiz aborted") {
		t.Fatalf("non-quiz cancellation reported as quiz abort:\n%s", out.String())
	}
}

func TestWatchLoopNavigationAtEnd(t *testing.T) {
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
	keyCh := make(chan byte, 2)
	keyCh <- 'n'
	keyCh <- 'q'
	tick := make(chan time.Time)
	var out bytes.Buffer
	if err := watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &out); err != nil {
		t.Fatalf("watchLoop: %v", err)
	}
	if !strings.Contains(out.String(), "no exercise in that direction") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestWatchLoopEscCancelsVerify(t *testing.T) {
	lib, st := oneExerciseLibrary(t)
	cfg, _ := config.LoadFrom("")
	started := make(chan struct{})
	sess := watch.New(watch.Options{
		Library: lib, State: st, Engine: &fakeWatchEngine{}, Reset: &fakeWatchReset{},
		Verify: func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		},
		Report: func([]assert.Result) {},
	})
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	keyCh := make(chan byte, 4)
	tick := make(chan time.Time)
	var out, errOut bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- watchLoop(context.Background(), sess, lib, cfg, "en", keyCh, tick, &out, &errOut) }()

	keyCh <- '\n' // start verification; it blocks until canceled
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("verification never started")
	}
	keyCh <- 0x1b // Esc cancels
	keyCh <- 'q'  // then quit

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("watchLoop: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watchLoop did not return after cancel/quit")
	}
	if !strings.Contains(out.String(), "verification canceled") {
		t.Fatalf("out = %q, want cancellation notice", out.String())
	}
	if strings.Contains(out.String(), "all exercises complete") {
		t.Fatalf("canceled verification must not complete the exercise: %q", out.String())
	}
}
