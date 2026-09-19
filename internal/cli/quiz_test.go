package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/watch"
)

func singleChoiceQuiz() *content.Exercise {
	return &content.Exercise{ID: "q1", Chapter: "c1", Type: content.TypeQuiz, Questions: []content.Question{
		{Prompt: "p", Options: []content.Option{
			{Text: "a", Correct: true, Explanation: "because"},
			{Text: "b"},
		}},
	}}
}

func feed(s string) chan byte {
	ch := make(chan byte, len(s)+1)
	for i := 0; i < len(s); i++ {
		ch <- s[i]
	}
	close(ch)
	return ch
}

func TestRunQuizSingleChoice(t *testing.T) {
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("1\n"), &out, singleChoiceQuiz(), "en")
	if err != nil {
		t.Fatalf("runQuiz: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestRunQuizRetriesUntilCorrect(t *testing.T) {
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("2\n1\n"), &out, singleChoiceQuiz(), "en")
	if err != nil {
		t.Fatalf("runQuiz: %v", err)
	}
	if !results[0].Passed || !bytes.Contains(out.Bytes(), []byte("because")) {
		t.Fatalf("results=%+v out=%s", results, out.String())
	}
}

func TestRunQuizAbort(t *testing.T) {
	ch := make(chan byte, 1)
	ch <- 0x03
	close(ch)
	var out bytes.Buffer
	if _, err := runQuiz(context.Background(), ch, &out, singleChoiceQuiz(), "en"); !errors.Is(err, watch.ErrQuizAborted) {
		t.Fatalf("err = %v, want ErrQuizAborted", err)
	}
}

func TestParseChoices(t *testing.T) {
	if got, err := parseChoices("1", 2, false); err != nil || len(got) != 1 || got[0] != 0 {
		t.Fatalf("single = %v, %v", got, err)
	}
	if got, err := parseChoices("1, 3", 3, true); err != nil || len(got) != 2 || got[0] != 0 || got[1] != 2 {
		t.Fatalf("multi = %v, %v", got, err)
	}
	for _, bad := range []string{"", "0", "9", "1,1", "a"} {
		if _, err := parseChoices(bad, 2, false); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
	if _, err := parseChoices("1,2", 3, false); err == nil {
		t.Fatal("single-choice must reject multiple")
	}
}

func multiSelectQuiz() *content.Exercise {
	return &content.Exercise{ID: "q1", Chapter: "c1", Type: content.TypeQuiz, Questions: []content.Question{
		{Prompt: "p", Multiple: true, Options: []content.Option{
			{Text: "a", Correct: true},
			{Text: "b"},
			{Text: "c", Correct: true},
		}},
	}}
}

func TestRunQuizMultiSelect(t *testing.T) {
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("1,3\n"), &out, multiSelectQuiz(), "en")
	if err != nil || !results[0].Passed {
		t.Fatalf("results=%+v err=%v", results, err)
	}
}

func TestRunQuizMultiSelectPartialRetry(t *testing.T) {
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("1\n1,3\n"), &out, multiSelectQuiz(), "en")
	if err != nil || !results[0].Passed {
		t.Fatalf("results=%+v err=%v", results, err)
	}
}

func TestReadQuizLineBackspaceAndEscape(t *testing.T) {
	// "2\x7f1\n" -> backspace removes '2' -> "1"; ESC[C (right arrow) ignored.
	var out bytes.Buffer
	line, err := readQuizLine(context.Background(), feed("2\x7f1\x1b[C\n"), &out, nil, "answer")
	if err != nil || line != "1" {
		t.Fatalf("line=%q err=%v", line, err)
	}
}

func TestReadQuizLineClosedNoNewlineAborts(t *testing.T) {
	var out bytes.Buffer
	_, err := readQuizLine(context.Background(), feed("1"), &out, nil, "answer")
	if !errors.Is(err, watch.ErrQuizAborted) {
		t.Fatalf("err=%v, want ErrQuizAborted", err)
	}
}

func TestRunQuizQuitKeyAborts(t *testing.T) {
	var out bytes.Buffer
	_, err := runQuiz(context.Background(), feed("q"), &out, singleChoiceQuiz(), "en")
	if !errors.Is(err, watch.ErrQuizAborted) {
		t.Fatalf("err = %v, want ErrQuizAborted", err)
	}
}

func TestRunQuizHintKeyThenAnswer(t *testing.T) {
	ex := singleChoiceQuiz()
	ex.Hints = []string{"UNIQUE_HINT_MARKER"}
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("h1\n"), &out, ex, "en")
	if err != nil {
		t.Fatalf("runQuiz: %v", err)
	}
	if !results[0].Passed || !bytes.Contains(out.Bytes(), []byte("UNIQUE_HINT_MARKER")) {
		t.Fatalf("hint not shown or not passed: %v\n%s", results, out.String())
	}
}

func TestReadQuizLineIgnoresLetters(t *testing.T) {
	var out bytes.Buffer
	line, err := readQuizLine(context.Background(), feed("a2x\n"), &out, nil, "answer")
	if err != nil || line != "2" {
		t.Fatalf("line=%q err=%v", line, err)
	}
	if bytes.ContainsAny(out.Bytes(), "ax") {
		t.Fatalf("letters were echoed: %q", out.String())
	}
}
