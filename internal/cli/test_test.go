package cli

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
)

func TestExitCodeForResults(t *testing.T) {
	if code := exitCodeForResults([]assert.Result{{Passed: true}}); code != 0 {
		t.Fatalf("all passed code = %d", code)
	}
	if code := exitCodeForResults([]assert.Result{{Passed: true}, {Passed: false}}); code != 1 {
		t.Fatalf("failure code = %d", code)
	}
}

func TestParseTestArgs(t *testing.T) {
	if _, err := parseTestArgs(nil, false); err == nil {
		t.Fatal("expected error when no id is given and --all is false")
	}
	id, err := parseTestArgs([]string{"c1/e1"}, false)
	if err != nil || id != "c1/e1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := parseTestArgs(nil, true); err != nil {
		t.Fatalf("--all without id: %v", err)
	}
	if _, err := parseTestArgs([]string{"c1/e1"}, true); err == nil {
		t.Fatal("expected error combining --all with an id")
	}
}

func TestSelectExercises(t *testing.T) {
	lib := sampleLibrary()
	if got := selectExercises(lib, "", true); len(got) != len(lib.Order) {
		t.Fatalf("all = %d", len(got))
	}
	ex := lib.Order[0]
	if got := selectExercises(lib, ex.FullID(), false); len(got) != 1 || got[0] != ex {
		t.Fatalf("single = %v", got)
	}
	if got := selectExercises(lib, "nope", false); got != nil {
		t.Fatalf("unknown = %v", got)
	}
}

func TestEngineNeededFor(t *testing.T) {
	quiz := &content.Exercise{ID: "q1", Type: content.TypeQuiz}
	ch := &content.Chapter{ID: "c1", Topology: "topology.clab.yml", TopologyPath: "topology.clab.yml"}
	quizTopo := &content.Exercise{ID: "q2", Type: content.TypeQuiz, ChapterRef: ch}
	lab := &content.Exercise{ID: "e1", Type: content.TypeConfigFix}
	if engineNeededFor([]*content.Exercise{quiz}, func(*content.Exercise) bool { return true }) {
		t.Fatal("skipped quiz must not need engine")
	}
	if engineNeededFor([]*content.Exercise{quiz}, func(*content.Exercise) bool { return false }) {
		t.Fatal("quiz must never need engine")
	}
	if engineNeededFor([]*content.Exercise{quizTopo}, func(*content.Exercise) bool { return false }) {
		t.Fatal("quiz with a topology chapter still must not need engine")
	}
	if !engineNeededFor([]*content.Exercise{lab}, func(*content.Exercise) bool { return false }) {
		t.Fatal("lab exercise needs engine")
	}
}

func TestTestCmdAnswersFlag(t *testing.T) {
	cmd := newTestCmd()
	f := cmd.Flags().Lookup("answers")
	if f == nil {
		t.Fatal("--answers flag is not registered")
	}
	if f.DefValue != "" {
		t.Fatalf("--answers default = %q", f.DefValue)
	}
}

func TestUnique(t *testing.T) {
	got := unique([]string{"a", "", "b", "a", ""})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unique = %v", got)
	}
}
