package watch

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
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
func (f *fakeEngine) Topology(context.Context, *content.Chapter) (*engine.Topology, error) {
	return &engine.Topology{}, nil
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
