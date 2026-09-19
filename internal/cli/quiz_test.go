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
	results, err := runQuiz(context.Background(), feed("1\n"), &out, singleChoiceQuiz())
	if err != nil {
		t.Fatalf("runQuiz: %v", err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestRunQuizRetriesUntilCorrect(t *testing.T) {
	var out bytes.Buffer
	results, err := runQuiz(context.Background(), feed("2\n1\n"), &out, singleChoiceQuiz())
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
	if _, err := runQuiz(context.Background(), ch, &out, singleChoiceQuiz()); !errors.Is(err, watch.ErrQuizAborted) {
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
