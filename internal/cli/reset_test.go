package cli

import (
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

func TestResetCommand(t *testing.T) {
	cmd := newResetCmd()
	if !strings.HasPrefix(cmd.Use, "reset") {
		t.Fatalf("Use = %q", cmd.Use)
	}
	if cmd.Flags().Lookup("hard") == nil {
		t.Fatal("missing --hard flag")
	}
}

func TestResetCmdRegistered(t *testing.T) {
	for _, c := range NewRootCmd().Commands() {
		if c.Name() == "reset" {
			return
		}
	}
	t.Fatal("reset command not registered")
}

func TestResolveExercise(t *testing.T) {
	ch := &content.Chapter{ID: "c1"}
	e1 := &content.Exercise{ID: "e1", Chapter: "c1", ChapterRef: ch}
	e2 := &content.Exercise{ID: "e2", Chapter: "c1", ChapterRef: ch}
	lib := &content.Library{Root: t.TempDir(), Order: []*content.Exercise{e1, e2},
		ByID: map[string]*content.Exercise{"c1/e1": e1, "c1/e2": e2}}
	st, err := state.Load(lib.Root)
	if err != nil {
		t.Fatal(err)
	}
	st.SetCurrent("c1/e2")
	if err := st.Save(lib.Root); err != nil {
		t.Fatal(err)
	}

	if got, err := resolveExercise(lib, "c1/e1"); err != nil || got != e1 {
		t.Fatalf("resolve full = %v, %v", got, err)
	}
	if got, err := resolveExercise(lib, "e1"); err != nil || got != e1 {
		t.Fatalf("resolve short = %v, %v", got, err)
	}
	if got, err := resolveExercise(lib, ""); err != nil || got != e2 {
		t.Fatalf("resolve current = %v, %v", got, err)
	}
	if _, err := resolveExercise(lib, "nope"); err == nil {
		t.Fatal("expected unknown error")
	}
}

func TestResolveExerciseNoCurrent(t *testing.T) {
	lib := &content.Library{Root: t.TempDir(), ByID: map[string]*content.Exercise{}}
	if _, err := resolveExercise(lib, ""); err == nil {
		t.Fatal("expected no-current error")
	}
}
