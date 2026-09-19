package watch

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

type fakeEngine struct{ deployed, destroyed []string }

func (f *fakeEngine) Deploy(_ context.Context, ch *content.Chapter) error {
	f.deployed = append(f.deployed, ch.ID)
	return nil
}
func (f *fakeEngine) Destroy(_ context.Context, ch *content.Chapter) error {
	f.destroyed = append(f.destroyed, ch.ID)
	return nil
}

type fakeReset struct{ fast, hard []string }

func (f *fakeReset) Fast(_ context.Context, ex *content.Exercise) error {
	f.fast = append(f.fast, ex.FullID())
	return nil
}
func (f *fakeReset) Hard(_ context.Context, ex *content.Exercise) error {
	f.hard = append(f.hard, ex.FullID())
	return nil
}

func twoChapters() *content.Library {
	ch1 := &content.Chapter{ID: "c1", TopologyName: "lab-c1"}
	e1 := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch1}
	e2 := &content.Exercise{ID: "e2", Chapter: "c1", ChapterRef: ch1}
	ch1.Exercises = []*content.Exercise{e1, e2}
	ch2 := &content.Chapter{ID: "c2", TopologyName: "lab-c2"}
	e3 := &content.Exercise{ID: "e3", Chapter: "c2", ChapterRef: ch2}
	ch2.Exercises = []*content.Exercise{e3}
	return &content.Library{
		Chapters: []*content.Chapter{ch1, ch2},
		Order:    []*content.Exercise{e1, e2, e3},
		ByID:     map[string]*content.Exercise{"c1/e1": e1, "c1/e2": e2, "c2/e3": e3},
	}
}

func TestSessionStartDeploysAndSeeds(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)

	var reported [][]assert.Result
	s := New(Options{
		Library: lib, State: st,
		Engine: fe, Reset: fr,
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func(r []assert.Result) { reported = append(reported, r) },
	})

	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(fe.deployed) != 1 || fe.deployed[0] != "c1" {
		t.Fatalf("deployed = %v", fe.deployed)
	}
	if len(fr.fast) != 1 || fr.fast[0] != "c1/e1" {
		t.Fatalf("fast = %v", fr.fast)
	}
	if s.Current().FullID() != "c1/e1" {
		t.Fatalf("current = %v", s.Current())
	}

	if err := s.VerifyNow(context.Background()); err != nil {
		t.Fatalf("VerifyNow: %v", err)
	}
	if len(reported) != 1 {
		t.Fatalf("reported = %v", reported)
	}
	if s.Current().FullID() != "c1/e2" {
		t.Fatalf("after pass should advance to c1/e2, got %v", s.Current())
	}
	if len(fr.fast) != 2 || fr.fast[1] != "c1/e2" {
		t.Fatalf("same-chapter advance should use fast reset, fast=%v", fr.fast)
	}
}

func TestSessionCrossChapterRedeploys(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{
		Library: lib, State: st, Engine: fe, Reset: fr,
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	_ = s.Start(context.Background())
	_ = s.VerifyNow(context.Background()) // c1/e1 -> c1/e2
	_ = s.VerifyNow(context.Background()) // c1/e2 -> c2/e3 (cross chapter)
	if len(fe.destroyed) != 1 || fe.destroyed[0] != "c1" {
		t.Fatalf("destroyed = %v", fe.destroyed)
	}
	if len(fe.deployed) != 2 || fe.deployed[1] != "c2" {
		t.Fatalf("deployed = %v", fe.deployed)
	}
	if s.Current().FullID() != "c2/e3" {
		t.Fatalf("current = %v", s.Current())
	}
}

func TestSessionVerifyFailKeepsCurrent(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: fe, Reset: fr,
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: false}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.VerifyNow(context.Background()); err != nil {
		t.Fatalf("VerifyNow: %v", err)
	}
	if s.Current().FullID() != "c1/e1" {
		t.Fatalf("current = %v", s.Current())
	}
	es := st.Get("c1/e1")
	if es.Status != state.StatusInProgress || es.Attempts != 1 {
		t.Fatalf("exercise state = %+v", es)
	}
	if len(fr.fast) != 1 {
		t.Fatalf("fast resets = %v", fr.fast)
	}
}

func TestSessionRunResolvesAndRejects(t *testing.T) {
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := s.Run(context.Background(), "e2"); err != nil {
		t.Fatalf("Run e2: %v", err)
	}
	if s.Current().FullID() != "c1/e2" {
		t.Fatalf("current = %v", s.Current())
	}
	if err := s.Run(context.Background(), "nope"); err == nil {
		t.Fatal("expected unknown exercise error")
	}

	ch1 := &content.Chapter{ID: "c1", TopologyName: "l1"}
	ch2 := &content.Chapter{ID: "c2", TopologyName: "l2"}
	d1 := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch1}
	d2 := &content.Exercise{ID: "e1", Chapter: "c2", ChapterRef: ch2}
	dup := &content.Library{Root: t.TempDir(), Order: []*content.Exercise{d1, d2},
		ByID: map[string]*content.Exercise{"c1/e1": d1, "c2/e1": d2}}
	dst, _ := state.Load(dup.Root)
	ds := New(Options{Library: dup, State: dst, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := ds.Run(context.Background(), "e1"); err == nil {
		t.Fatal("expected duplicate ID error")
	}
}

func TestSessionResumesFromState(t *testing.T) {
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	st.SetCurrent("c1/e2")
	s := New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if s.Current().FullID() != "c1/e2" {
		t.Fatalf("current = %v", s.Current())
	}
}

func TestSessionCompletionClearsCurrent(t *testing.T) {
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := s.VerifyNow(context.Background()); err != nil {
			t.Fatalf("VerifyNow %d: %v", i, err)
		}
	}
	if s.Current() != nil {
		t.Fatalf("current = %v, want nil after completion", s.Current())
	}
	if st.Current != "" {
		t.Fatalf("state current = %q, want empty", st.Current)
	}
	if err := s.VerifyNow(context.Background()); err == nil {
		t.Fatal("expected error verifying with no active exercise")
	}
}

func TestSessionEmptyLibrary(t *testing.T) {
	lib := &content.Library{Root: t.TempDir(), ByID: map[string]*content.Exercise{}}
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := s.Start(context.Background()); err == nil {
		t.Fatal("expected empty library error")
	}
}

func TestSessionBusyGuard(t *testing.T) {
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	var s *Session
	s = New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) {
			if err := s.VerifyNow(context.Background()); err != nil {
				t.Errorf("re-entrant VerifyNow: %v", err)
			}
			return []assert.Result{{Passed: true}}, nil
		},
		Report: func([]assert.Result) {},
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.VerifyNow(context.Background()); err != nil {
		t.Fatalf("VerifyNow: %v", err)
	}
}

func TestSessionStatusResetAndCleanup(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	var statuses []string
	s := New(Options{
		Library: lib, State: st, Engine: fe, Reset: fr,
		Status: func(m string) { statuses = append(statuses, m) },
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := s.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := s.ResetCurrent(context.Background(), true); err != nil {
		t.Fatalf("ResetCurrent: %v", err)
	}
	if len(fr.hard) != 1 || fr.hard[0] != "c1/e1" {
		t.Fatalf("hard = %v", fr.hard)
	}
	if err := s.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if len(fe.destroyed) != 2 {
		t.Fatalf("destroyed = %v", fe.destroyed)
	}
	joined := strings.Join(statuses, "\n")
	for _, want := range []string{
		"deploying chapter c1", "chapter c1 ready",
		"resetting exercise c1/e1 (hard)", "reset complete",
		"destroying chapter c1", "destroying chapter c2",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("status missing %q:\n%s", want, joined)
		}
	}
}

func TestSessionResetCurrentFast(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: fe, Reset: fr,
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil }})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetCurrent(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(fr.fast) != 2 || fr.fast[1] != "c1/e1" {
		t.Fatalf("fast = %v", fr.fast)
	}
}

func TestSessionResetCurrentNil(t *testing.T) {
	lib := &content.Library{Root: t.TempDir(), ByID: map[string]*content.Exercise{}}
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: &fakeEngine{}, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil }})
	if err := s.ResetCurrent(context.Background(), true); err != nil {
		t.Fatalf("nil current: %v", err)
	}
}

type partialFailEngine struct {
	fakeEngine
	failOn string
}

func (p *partialFailEngine) Destroy(_ context.Context, ch *content.Chapter) error {
	p.destroyed = append(p.destroyed, ch.ID)
	if ch.ID == p.failOn {
		return errors.New("boom")
	}
	return nil
}

func TestSessionCleanupBestEffort(t *testing.T) {
	fe := &partialFailEngine{failOn: "c1"}
	lib := twoChapters()
	lib.Root = t.TempDir()
	st, _ := state.Load(lib.Root)
	s := New(Options{Library: lib, State: st, Engine: fe, Reset: &fakeReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil }})
	if err := s.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	err := s.Cleanup(context.Background())
	if err == nil || !strings.Contains(err.Error(), "c1") {
		t.Fatalf("err = %v, want chapter c1 failure", err)
	}
	if len(fe.destroyed) != 2 {
		t.Fatalf("destroyed = %v, want both chapters attempted", fe.destroyed)
	}
}
