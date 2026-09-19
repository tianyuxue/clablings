package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/watch"
)

// errQuizHelp signals that the user asked for help (h/c) and the question
// should be redrawn.
var errQuizHelp = errors.New("quiz help requested")

// runQuiz interactively asks each question until every answer is correct and
// returns the final (all-correct) grading results. loc selects the language for
// the `c` concept help.
func runQuiz(ctx context.Context, in <-chan byte, out io.Writer, ex *content.Exercise, loc string) ([]assert.Result, error) {
	help := func(b byte) bool {
		switch b {
		case 'h', 'H':
			watch.WriteHints(out, ex)
			return true
		case 'c', 'C':
			if ex.ChapterRef != nil {
				_ = printChapterRead(out, ex.ChapterRef, loc)
			}
			return true
		}
		return false
	}
	fmt.Fprintln(out, "(enter option numbers; h hint, c concept, q quit)")
	answers := make([][]int, len(ex.Questions))
	for qi := range ex.Questions {
		q := &ex.Questions[qi]
		for {
			fmt.Fprintf(out, "\nQ%d: %s\n", qi+1, q.Prompt)
			for oi := range q.Options {
				fmt.Fprintf(out, "  %d) %s\n", oi+1, q.Options[oi].Text)
			}
			prompt := "answer"
			if q.Multiple {
				prompt = "answers (comma-separated)"
			}
			fmt.Fprintf(out, "%s> ", prompt)
			line, err := readQuizLine(ctx, in, out, help)
			if errors.Is(err, errQuizHelp) {
				continue // redraw the question and prompt
			}
			if err != nil {
				return nil, err
			}
			chosen, perr := parseChoices(line, len(q.Options), q.Multiple)
			if perr != nil {
				fmt.Fprintf(out, "invalid input: %v\n", perr)
				continue
			}
			if sameChoiceSet(chosen, correctChoices(q)) {
				answers[qi] = chosen
				break
			}
			fmt.Fprintln(out, "not quite:")
			printExplanations(out, q, chosen)
		}
	}
	return assert.GradeQuiz(ex, answers), nil
}

// readQuizLine reads one line, echoing only answer characters (digits, commas,
// spaces) until Enter. Ctrl+C, q, a closed input stream, or a cancelled context
// aborts. h/c are passed to help (when non-nil) and return errQuizHelp so the
// caller redraws the question. Other bytes are ignored; ANSI CSI sequences are
// skipped.
func readQuizLine(ctx context.Context, in <-chan byte, out io.Writer, help func(byte) bool) (string, error) {
	var buf []byte
	esc := false // saw ESC
	csi := false // inside ESC [ ... sequence
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case b, ok := <-in:
			if !ok {
				return "", watch.ErrQuizAborted
			}
			if csi {
				if b >= 0x40 && b <= 0x7e { // final byte of CSI
					csi = false
				}
				continue
			}
			if esc {
				esc = false
				if b == '[' {
					csi = true
				}
				continue
			}
			if b == 'q' || b == 'Q' {
				fmt.Fprintln(out)
				return "", watch.ErrQuizAborted
			}
			if help != nil && (b == 'h' || b == 'H' || b == 'c' || b == 'C') {
				if help(b) {
					return "", errQuizHelp
				}
				continue
			}
			switch b {
			case 0x1b:
				esc = true
			case 0x03:
				fmt.Fprintln(out)
				return "", watch.ErrQuizAborted
			case '\r', '\n':
				fmt.Fprintln(out)
				return string(buf), nil
			case 0x7f, 0x08:
				if len(buf) > 0 {
					buf = buf[:len(buf)-1]
					fmt.Fprint(out, "\b \b")
				}
			default:
				if (b >= '0' && b <= '9') || b == ',' || b == ' ' {
					buf = append(buf, b)
					fmt.Fprintf(out, "%c", b)
				}
			}
		}
	}
}

func parseChoices(line string, n int, multiple bool) ([]int, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("choose an option")
	}
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.Atoi(part)
		if err != nil || v < 1 || v > n {
			return nil, fmt.Errorf("expected a number between 1 and %d", n)
		}
		idx := v - 1
		if seen[idx] {
			return nil, fmt.Errorf("duplicate choice %d", v)
		}
		seen[idx] = true
		out = append(out, idx)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("choose an option")
	}
	if !multiple && len(out) != 1 {
		return nil, fmt.Errorf("choose exactly one option")
	}
	sort.Ints(out)
	return out, nil
}

func correctChoices(q *content.Question) []int {
	var out []int
	for i := range q.Options {
		if q.Options[i].Correct {
			out = append(out, i)
		}
	}
	return out
}

func sameChoiceSet(a, b []int) bool {
	sa := append([]int(nil), a...)
	sb := append([]int(nil), b...)
	sort.Ints(sa)
	sort.Ints(sb)
	if len(sa) != len(sb) {
		return false
	}
	for i := range sa {
		if sa[i] != sb[i] {
			return false
		}
	}
	return true
}

func printExplanations(out io.Writer, q *content.Question, chosen []int) {
	chosenSet := map[int]bool{}
	for _, c := range chosen {
		chosenSet[c] = true
	}
	for i := range q.Options {
		o := &q.Options[i]
		if !o.Correct && !chosenSet[i] {
			continue
		}
		mark := " "
		if o.Correct {
			mark = "*"
		}
		fmt.Fprintf(out, "  %s %d) %s", mark, i+1, o.Text)
		if o.Explanation != "" {
			fmt.Fprintf(out, " — %s", o.Explanation)
		}
		fmt.Fprintln(out)
	}
}
