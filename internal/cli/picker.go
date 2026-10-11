package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

const pickerTitle = "Select an exercise  (↑/↓ move · Enter to jump · Esc cancel)"

// pickerRow is one line in the jump picker: a chapter header or an exercise.
type pickerRow struct {
	header    bool
	chapterID string
	title     string
	ex        *content.Exercise
	current   bool
	status    state.Status
}

// buildRows flattens the library into chapter headers and exercise rows.
func buildRows(lib *content.Library, currentID string, statusOf func(string) state.Status) []pickerRow {
	var rows []pickerRow
	for _, ch := range lib.Chapters {
		if len(ch.Exercises) == 0 {
			continue
		}
		rows = append(rows, pickerRow{header: true, chapterID: ch.ID, title: ch.Title})
		for _, ex := range ch.Exercises {
			rows = append(rows, pickerRow{
				ex:      ex,
				current: ex.FullID() == currentID,
				status:  statusOf(ex.FullID()),
			})
		}
	}
	return rows
}

// initialCursor returns the index of the current exercise, or the first
// selectable row when currentID is absent.
func initialCursor(rows []pickerRow, currentID string) int {
	first := -1
	for i, r := range rows {
		if r.header {
			continue
		}
		if first < 0 {
			first = i
		}
		if r.ex.FullID() == currentID {
			return i
		}
	}
	if first < 0 {
		return 0
	}
	return first
}

// moveCursor moves by delta, skipping chapter headers and clamping to the
// first/last selectable row.
func moveCursor(rows []pickerRow, cursor, delta int) int {
	if len(rows) == 0 || delta == 0 {
		return cursor
	}
	i := cursor
	for {
		next := i + delta
		if next < 0 || next >= len(rows) {
			return cursor
		}
		i = next
		if !rows[i].header {
			return i
		}
	}
}

func statusMark(s state.Status) string {
	switch s {
	case state.StatusPassed:
		return "√"
	case state.StatusInProgress:
		return "›"
	default:
		return "·"
	}
}

func typeBadge(t content.ExerciseType) string {
	switch t {
	case content.TypeQuiz:
		return "quiz"
	case content.TypeCodeImplement:
		return "code"
	default:
		return "fix"
	}
}

// Note: counts runes, not terminal cells; wide (CJK) runes may occupy two
// columns, so a line can still wrap. Acceptable for v1.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	return string([]rune(s)[:width-1]) + "…"
}

func rowText(r pickerRow, isCursor bool, width int) string {
	cur := " "
	if isCursor {
		cur = ">"
	}
	var s string
	if r.header {
		s = truncate(fmt.Sprintf("%s   ── %s  %s", cur, r.chapterID, r.title), width)
	} else {
		lead := " "
		if r.current {
			lead = "▌"
		}
		s = truncate(fmt.Sprintf("%s%s %s  %s  %s  [%s]", cur, lead, statusMark(r.status), r.ex.ID, r.ex.Title, typeBadge(r.ex.Type)), width)
	}
	if isCursor {
		return reverse(s)
	}
	return s
}

func exerciseCount(rows []pickerRow) int {
	n := 0
	for _, r := range rows {
		if !r.header {
			n++
		}
	}
	return n
}

// pickerLines returns exactly `height` display lines (title + body) for the
// current cursor position, windowed around the cursor. Long lists reserve one
// line each at top and bottom for overflow indicators so the cursor row always
// fits.
func pickerLines(rows []pickerRow, cursor, width, height int) []string {
	if width < 24 {
		width = 80
	}
	if height < 4 {
		height = 24
	}
	body := height - 1
	out := make([]string, 0, height)
	out = append(out, truncate(fmt.Sprintf("%s  (%d exercises)", pickerTitle, exerciseCount(rows)), width))

	if len(rows) <= body {
		for i := 0; i < len(rows); i++ {
			out = append(out, rowText(rows[i], i == cursor, width))
		}
		for len(out) < height {
			out = append(out, "")
		}
		return out
	}

	content := body - 2
	if content < 1 {
		content = 1
	}
	start := cursor - content/2
	maxStart := len(rows) - content
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	if start < 0 {
		start = 0
	}
	end := start + content
	if end > len(rows) {
		end = len(rows)
	}
	if start > 0 {
		out = append(out, fmt.Sprintf("  ↑ %d more", start))
	} else {
		out = append(out, "")
	}
	for i := start; i < end; i++ {
		out = append(out, rowText(rows[i], i == cursor, width))
	}
	if end < len(rows) {
		out = append(out, fmt.Sprintf("  ↓ %d more", len(rows)-end))
	} else {
		out = append(out, "")
	}
	return out
}

// pickerSize returns the terminal width/height, falling back to 80x24.
func pickerSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 80, 24
	}
	return w, h
}

// runPicker renders the jump picker and returns the chosen exercise FullID, or
// "" when the user cancels (Esc / Ctrl+C / q / closed input).
func runPicker(ctx context.Context, in <-chan byte, out io.Writer, lib *content.Library, currentID string, statusOf func(string) state.Status) (string, error) {
	rows := buildRows(lib, currentID, statusOf)
	if len(rows) == 0 {
		return "", nil
	}
	cursor := initialCursor(rows, currentID)
	width, height := pickerSize()
	lines := pickerLines(rows, cursor, width, height)
	drawn := 0
	color := colorOn()

	draw := func() {
		if !color && drawn > 0 {
			return // non-TTY: render once, then stay static
		}
		if color && drawn > 0 {
			fmt.Fprintf(out, "\x1b[%dA", drawn)
		}
		for _, ln := range lines {
			if color {
				fmt.Fprint(out, "\r\x1b[K")
			}
			fmt.Fprintln(out, ln)
		}
		drawn = len(lines)
	}
	clear := func() {
		if !color || drawn == 0 {
			return
		}
		fmt.Fprintf(out, "\x1b[%dA", drawn)
		for i := 0; i < drawn; i++ {
			fmt.Fprint(out, "\r\x1b[K\n")
		}
		fmt.Fprintf(out, "\x1b[%dA", drawn)
	}
	if color {
		fmt.Fprint(out, "\x1b[?25l") // hide cursor
		defer fmt.Fprint(out, "\x1b[?25h")
	}
	defer clear()
	draw()

	move := func(delta int) {
		cursor = moveCursor(rows, cursor, delta)
		lines = pickerLines(rows, cursor, width, height)
		draw()
	}

	handle := func(b byte) (sel string, done bool) {
		switch b {
		case '\r', '\n':
			return rows[cursor].ex.FullID(), true
		case 0x03, 'q', 'Q':
			return "", true
		case 'j', 'J':
			move(1)
		case 'k', 'K':
			move(-1)
		}
		return "", false
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case b, ok := <-in:
			if !ok {
				return "", nil
			}
			if b == 0x1b {
				final, follow, cancel := readCSI(in)
				if cancel {
					return "", nil
				}
				if follow != 0 {
					if sel, done := handle(follow); done {
						return sel, nil
					}
					continue
				}
				switch final {
				case 'A':
					move(-1)
				case 'B':
					move(1)
				}
				continue
			}
			if sel, done := handle(b); done {
				return sel, nil
			}
		}
	}
}

// readCSI reads the bytes after ESC. It returns the final byte of an ANSI CSI
// sequence (e.g. 'A'/'B' for up/down), or follow (with final 0) when a non-CSI
// byte immediately trailed ESC, or cancel=true when ESC stood alone.
func readCSI(in <-chan byte) (final byte, follow byte, cancel bool) {
	b, ok := recvByte(in, 50*time.Millisecond)
	if !ok {
		return 0, 0, true
	}
	if b != '[' {
		return 0, b, false
	}
	for {
		fb, ok := recvByte(in, 50*time.Millisecond)
		if !ok {
			return 0, 0, false
		}
		if fb >= 0x40 && fb <= 0x7e {
			return fb, 0, false
		}
	}
}
