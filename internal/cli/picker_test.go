package cli

import (
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

func pickerLib() *content.Library {
	c1 := &content.Chapter{ID: "c1", Title: "Chapter One"}
	e1 := &content.Exercise{ID: "e1", Chapter: "c1", Title: "First", Type: content.TypeQuiz, ChapterRef: c1}
	e2 := &content.Exercise{ID: "e2", Chapter: "c1", Title: "Second", Type: content.TypeConfigFix, ChapterRef: c1}
	c1.Exercises = []*content.Exercise{e1, e2}
	c2 := &content.Chapter{ID: "c2", Title: "Chapter Two"}
	e3 := &content.Exercise{ID: "e3", Chapter: "c2", Title: "Third", Type: content.TypeCodeImplement, ChapterRef: c2}
	c2.Exercises = []*content.Exercise{e3}
	return &content.Library{
		Chapters: []*content.Chapter{c1, c2},
		Order:    []*content.Exercise{e1, e2, e3},
		ByID:     map[string]*content.Exercise{"c1/e1": e1, "c1/e2": e2, "c2/e3": e3},
	}
}

func pickerStatus(m map[string]state.Status) func(string) state.Status {
	return func(id string) state.Status { return m[id] }
}

func TestBuildRowsGroupsByChapter(t *testing.T) {
	lib := pickerLib()
	rows := buildRows(lib, "c1/e2", pickerStatus(map[string]state.Status{"c1/e1": state.StatusPassed}))
	if len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
	wantHeader := []bool{true, false, false, true, false}
	for i, h := range wantHeader {
		if rows[i].header != h {
			t.Fatalf("row %d header = %v, want %v", i, rows[i].header, h)
		}
	}
	if rows[0].chapterID != "c1" || rows[3].chapterID != "c2" {
		t.Fatalf("headers = %q,%q", rows[0].chapterID, rows[3].chapterID)
	}
	if rows[1].status != state.StatusPassed || rows[2].status != state.StatusTodo {
		t.Fatalf("statuses = %q,%q", rows[1].status, rows[2].status)
	}
	if !rows[2].current {
		t.Fatal("row 2 should be current (c1/e2)")
	}
}

func TestInitialCursor(t *testing.T) {
	lib := pickerLib()
	rows := buildRows(lib, "c2/e3", pickerStatus(nil))
	if got := initialCursor(rows, "c2/e3"); got != 4 {
		t.Fatalf("initialCursor = %d, want 4", got)
	}
	if got := initialCursor(rows, "missing"); got != 1 {
		t.Fatalf("initialCursor fallback = %d, want 1 (first exercise)", got)
	}
}

func TestMoveCursorSkipsHeaders(t *testing.T) {
	lib := pickerLib()
	rows := buildRows(lib, "", pickerStatus(nil)) // rows: 0 hdr,1 e1,2 e2,3 hdr,4 e3
	if got := moveCursor(rows, 2, 1); got != 4 {
		t.Fatalf("down from e2 = %d, want 4 (skip header)", got)
	}
	if got := moveCursor(rows, 4, -1); got != 2 {
		t.Fatalf("up from e3 = %d, want 2 (skip header)", got)
	}
	if got := moveCursor(rows, 4, 1); got != 4 {
		t.Fatalf("down at end = %d, want 4 (clamp)", got)
	}
	if got := moveCursor(rows, 1, -1); got != 1 {
		t.Fatalf("up at start = %d, want 1 (clamp)", got)
	}
}

func TestPickerLinesShowMarkers(t *testing.T) {
	lib := pickerLib()
	rows := buildRows(lib, "c1/e2", pickerStatus(map[string]state.Status{"c1/e1": state.StatusPassed}))
	lines := pickerLines(rows, 2, 80, 24)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Select an exercise") {
		t.Fatalf("missing title:\n%s", joined)
	}
	if !strings.Contains(joined, "√") {
		t.Fatalf("missing passed marker:\n%s", joined)
	}
	if !strings.Contains(joined, "▌") {
		t.Fatalf("missing current marker:\n%s", joined)
	}
	if !strings.Contains(joined, ">") {
		t.Fatalf("missing cursor marker:\n%s", joined)
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Fatalf("short = %q", got)
	}
	if got := truncate("hello", 4); got != "hel…" {
		t.Fatalf("long = %q", got)
	}
}
