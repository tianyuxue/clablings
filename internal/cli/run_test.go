package cli

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
	"github.com/tianyuxue/clablings/internal/watch"
)

type stubEngine struct{}

func (stubEngine) Deploy(context.Context, *content.Chapter) error  { return nil }
func (stubEngine) Destroy(context.Context, *content.Chapter) error { return nil }

type stubReset struct{}

func (stubReset) Fast(context.Context, *content.Exercise) error { return nil }
func (stubReset) Hard(context.Context, *content.Exercise) error { return nil }

func TestSessionRunResolvesShortID(t *testing.T) {
	ch := &content.Chapter{ID: "c1", TopologyName: "lab"}
	e1 := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch}
	ch.Exercises = []*content.Exercise{e1}
	lib := &content.Library{
		Root:     t.TempDir(),
		Chapters: []*content.Chapter{ch},
		Order:    []*content.Exercise{e1},
		ByID:     map[string]*content.Exercise{"c1/e1": e1},
	}
	st, _ := state.Load(lib.Root)
	s := watch.New(watch.Options{
		Library: lib, State: st, Engine: stubEngine{}, Reset: stubReset{},
		Verify: func(context.Context, *content.Exercise) ([]assert.Result, error) { return nil, nil },
	})
	if err := s.Run(context.Background(), "e1"); err != nil {
		t.Fatalf("Run short id: %v", err)
	}
	if s.Current() == nil || s.Current().FullID() != "c1/e1" {
		t.Fatalf("current = %v", s.Current())
	}
}

func TestRunCmdRegistered(t *testing.T) {
	cmd := NewRootCmd()
	for _, c := range cmd.Commands() {
		if c.Name() == "run" {
			return
		}
	}
	t.Fatal("run command not registered")
}
