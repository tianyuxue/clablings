package cli

import (
	"bytes"
	"context"
	"fmt"
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
	return func(id string) state.Status {
		if s, ok := m[id]; ok {
			return s
		}
		return state.StatusTodo
	}
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
	if got := moveCursor(rows, 2, 0); got != 2 {
		t.Fatalf("delta 0 = %d, want 2 (unchanged)", got)
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
	if !strings.Contains(joined, "[passed]") {
		t.Fatalf("missing passed status tag:\n%s", joined)
	}
	if !strings.Contains(joined, "[todo]") {
		t.Fatalf("missing todo status tag:\n%s", joined)
	}
	if !strings.Contains(joined, "●") {
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

func longPickerLib(n int) *content.Library {
	ch := &content.Chapter{ID: "c1", Title: "Big"}
	var exs []*content.Exercise
	byID := map[string]*content.Exercise{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("e%03d", i)
		ex := &content.Exercise{ID: id, Chapter: "c1", Title: fmt.Sprintf("Ex %d", i), Type: content.TypeConfigFix, ChapterRef: ch}
		exs = append(exs, ex)
		byID["c1/"+id] = ex
	}
	ch.Exercises = exs
	return &content.Library{Chapters: []*content.Chapter{ch}, Order: exs, ByID: byID}
}

func TestPickerLinesViewport(t *testing.T) {
	// short list: all rows shown, padded to exactly height lines
	short := pickerLib()
	ls := pickerLines(buildRows(short, "c1/e1", pickerStatus(nil)), 1, 80, 24)
	if len(ls) != 24 {
		t.Fatalf("short len = %d, want 24", len(ls))
	}
	if !strings.Contains(strings.Join(ls, "\n"), "e2") {
		t.Fatalf("short list should show all rows:\n%s", strings.Join(ls, "\n"))
	}

	// long list, cursor at the last row: cursor must stay visible
	long := longPickerLib(100)
	lrows := buildRows(long, "c1/e099", pickerStatus(nil))
	lcursor := initialCursor(lrows, "c1/e099")
	ls = pickerLines(lrows, lcursor, 80, 24)
	if len(ls) != 24 {
		t.Fatalf("long len = %d, want 24", len(ls))
	}
	joined := strings.Join(ls, "\n")
	if !strings.Contains(joined, ">") || !strings.Contains(joined, "e099") {
		t.Fatalf("cursor row must be visible at the bottom:\n%s", joined)
	}
	if !strings.Contains(joined, "↑ ") {
		t.Fatalf("expected an up indicator for a bottom window:\n%s", joined)
	}

	// tiny height still returns exactly height lines with the cursor visible
	ls = pickerLines(lrows, lcursor, 80, 4)
	if len(ls) != 4 {
		t.Fatalf("tiny len = %d, want 4", len(ls))
	}
	if !strings.Contains(strings.Join(ls, "\n"), ">") {
		t.Fatalf("cursor must be visible at tiny height:\n%s", strings.Join(ls, "\n"))
	}

	// cursor at the very top of a long list: no up indicator
	ls = pickerLines(lrows, 0, 80, 24)
	if strings.Contains(strings.Join(ls, "\n"), "↑ ") {
		t.Fatalf("top window should have no up indicator:\n%s", strings.Join(ls, "\n"))
	}
}

func feedBytes(bs ...byte) chan byte {
	ch := make(chan byte, len(bs)+4)
	for _, b := range bs {
		ch <- b
	}
	return ch
}

func TestRunPickerSelectsWithArrows(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	// ESC [ B (down) twice, then Enter.
	in := feedBytes(0x1b, '[', 'B', 0x1b, '[', 'B', '\r')
	got, err := runPicker(context.Background(), in, &out, lib, "c1/e1", pickerStatus(nil))
	if err != nil {
		t.Fatalf("runPicker: %v", err)
	}
	// c1/e1 -> down -> c1/e2 -> down -> c2/e3 (header skipped)
	if got != "c2/e3" {
		t.Fatalf("selected = %q, want c2/e3", got)
	}
}

func TestRunPickerJKMove(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	in := feedBytes('j', '\r') // down one from c1/e1 -> c1/e2
	got, _ := runPicker(context.Background(), in, &out, lib, "c1/e1", pickerStatus(nil))
	if got != "c1/e2" {
		t.Fatalf("selected = %q, want c1/e2", got)
	}
}

func TestRunPickerEscCancels(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	in := feedBytes(0x1b) // bare ESC (no following bytes; recvByte times out)
	got, err := runPicker(context.Background(), in, &out, lib, "c1/e1", pickerStatus(nil))
	if err != nil {
		t.Fatalf("runPicker: %v", err)
	}
	if got != "" {
		t.Fatalf("esc should cancel, got %q", got)
	}
}

func TestRunPickerQCancels(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	in := feedBytes('q')
	got, _ := runPicker(context.Background(), in, &out, lib, "c1/e1", pickerStatus(nil))
	if got != "" {
		t.Fatalf("q should cancel, got %q", got)
	}
}

func closedChan() chan byte {
	ch := make(chan byte)
	close(ch)
	return ch
}

func TestRunPickerEscThenKeyNavigates(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	in := feedBytes(0x1b, 'j', '\r') // ESC immediately followed by j: j must be reprocessed
	got, _ := runPicker(context.Background(), in, &out, lib, "c1/e1", pickerStatus(nil))
	if got != "c1/e2" {
		t.Fatalf("selected = %q, want c1/e2", got)
	}
}

func TestRunPickerArrowUp(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	in := feedBytes(0x1b, '[', 'A', 0x1b, '[', 'A', '\r') // up twice from c2/e3 -> c1/e1
	got, _ := runPicker(context.Background(), in, &out, lib, "c2/e3", pickerStatus(nil))
	if got != "c1/e1" {
		t.Fatalf("selected = %q, want c1/e1", got)
	}
}

func TestRunPickerCtrlCCancels(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	got, _ := runPicker(context.Background(), feedBytes(0x03), &out, lib, "c1/e1", pickerStatus(nil))
	if got != "" {
		t.Fatalf("ctrl-c should cancel, got %q", got)
	}
}

func TestRunPickerClosedInput(t *testing.T) {
	lib := pickerLib()
	var out bytes.Buffer
	got, err := runPicker(context.Background(), closedChan(), &out, lib, "c1/e1", pickerStatus(nil))
	if err != nil || got != "" {
		t.Fatalf("closed input => (%q,%v), want empty,nil", got, err)
	}
}
