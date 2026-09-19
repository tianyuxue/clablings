package assert

import (
	"fmt"
	"sort"

	"github.com/tianyuxue/clablings/internal/content"
)

// GradeQuiz grades a quiz exercise against per-question chosen option indices.
func GradeQuiz(ex *content.Exercise, answers [][]int) []Result {
	if len(answers) != len(ex.Questions) {
		return []Result{{
			Exercise: ex.FullID(), Name: ex.Title, Type: "quiz", Layer: "control-plane",
			Expected: fmt.Sprintf("%d answers", len(ex.Questions)),
			Actual:   fmt.Sprintf("%d answers provided", len(answers)),
			Passed:   false,
		}}
	}
	results := make([]Result, 0, len(ex.Questions))
	for qi, q := range ex.Questions {
		passed := sameSet(answers[qi], correctIndices(q))
		actual := "wrong options"
		if passed {
			actual = "correct"
		}
		results = append(results, Result{
			Exercise: ex.FullID(), Name: q.Prompt, Type: "quiz", Layer: "control-plane",
			Expected: "the correct option set", Actual: actual, Passed: passed,
		})
	}
	return results
}

func correctIndices(q content.Question) []int {
	var out []int
	for i, o := range q.Options {
		if o.Correct {
			out = append(out, i)
		}
	}
	return out
}

func sameSet(a, b []int) bool {
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
