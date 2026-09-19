package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAnswers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "answers.yaml")
	if err := os.WriteFile(path, []byte("c1/q1:\n  - [0]\n  - [0, 2]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a, err := LoadAnswers(path)
	if err != nil {
		t.Fatalf("LoadAnswers: %v", err)
	}
	got := a["c1/q1"]
	if len(got) != 2 || len(got[0]) != 1 || got[0][0] != 0 || len(got[1]) != 2 || got[1][1] != 2 {
		t.Fatalf("answers = %+v", a)
	}
}

func TestLoadAnswersMissingFile(t *testing.T) {
	if _, err := LoadAnswers(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}
