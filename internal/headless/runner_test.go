package headless

import (
	"context"
	"slices"
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/watch"
)

type fakeEngine struct {
	deployed, destroyed []string
}

func (f *fakeEngine) Deploy(_ context.Context, ch *content.Chapter) error {
	f.deployed = append(f.deployed, ch.ID)
	return nil
}
func (f *fakeEngine) Destroy(_ context.Context, ch *content.Chapter) error {
	f.destroyed = append(f.destroyed, ch.ID)
	return nil
}

type fakeReset struct{ hard []string }

func (f *fakeReset) Fast(context.Context, *content.Exercise) error { return nil }
func (f *fakeReset) Hard(_ context.Context, ex *content.Exercise) error {
	f.hard = append(f.hard, ex.FullID())
	return nil
}

func library() *content.Library {
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

var _ watch.Engine = (*fakeEngine)(nil)
var _ watch.Resetter = (*fakeReset)(nil)

func TestRunAllDeploysPerChapterAndHardResets(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	var got []string
	err := RunAll(context.Background(), library(), fe, fr, func(_ context.Context, ex *content.Exercise) ([]assert.Result, error) {
		got = append(got, ex.FullID())
		return []assert.Result{{Passed: ex.ID != "e2"}}, nil
	}, func(id string, results []assert.Result) {
		got = append(got, "report:"+id)
	})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if len(fe.deployed) != 2 || fe.deployed[0] != "c1" || fe.deployed[1] != "c2" {
		t.Fatalf("deployed = %v", fe.deployed)
	}
	if len(fe.destroyed) != 2 || fe.destroyed[0] != "c1" || fe.destroyed[1] != "c2" {
		t.Fatalf("destroyed = %v", fe.destroyed)
	}
	if len(fr.hard) != 3 {
		t.Fatalf("hard resets = %v", fr.hard)
	}
	if got[1] != "report:c1/e1" || !slices.Contains(got, "report:c1/e2") || !slices.Contains(got, "report:c2/e3") {
		t.Fatalf("report order = %v", got)
	}
}

func TestRunOneDeploysResetsVerifiesAndDestroys(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeReset{}
	var verified string
	results, err := RunOne(context.Background(), library(), "c2/e3", fe, fr, func(_ context.Context, ex *content.Exercise) ([]assert.Result, error) {
		verified = ex.FullID()
		return []assert.Result{{Passed: true}}, nil
	})
	if err != nil {
		t.Fatalf("RunOne: %v", err)
	}
	if verified != "c2/e3" || len(results) != 1 || !results[0].Passed {
		t.Fatalf("verified=%q results=%v", verified, results)
	}
	if len(fe.deployed) != 1 || fe.deployed[0] != "c2" {
		t.Fatalf("deployed = %v", fe.deployed)
	}
	if len(fe.destroyed) != 1 || fe.destroyed[0] != "c2" {
		t.Fatalf("destroyed = %v", fe.destroyed)
	}
	if len(fr.hard) != 1 || fr.hard[0] != "c2/e3" {
		t.Fatalf("hard resets = %v", fr.hard)
	}
}

func TestRunOneUnknownExercise(t *testing.T) {
	_, err := RunOne(context.Background(), library(), "nope", &fakeEngine{}, &fakeReset{}, nil)
	if err == nil {
		t.Fatal("expected error for unknown exercise")
	}
}
