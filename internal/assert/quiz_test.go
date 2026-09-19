package assert

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
)

func sampleQuiz() *content.Exercise {
	return &content.Exercise{ID: "q1", Chapter: "c1", Type: content.TypeQuiz, Questions: []content.Question{
		{Prompt: "q1", Options: []content.Option{{Correct: true}, {}}},
		{Prompt: "q2", Multiple: true, Options: []content.Option{{Correct: true}, {}, {Correct: true}}},
	}}
}

func TestGradeQuizPass(t *testing.T) {
	results := GradeQuiz(sampleQuiz(), [][]int{{0}, {0, 2}})
	if len(results) != 2 || !results[0].Passed || !results[1].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestGradeQuizFail(t *testing.T) {
	results := GradeQuiz(sampleQuiz(), [][]int{{1}, {0, 2}})
	if results[0].Passed || !results[1].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestGradeQuizCountMismatch(t *testing.T) {
	results := GradeQuiz(sampleQuiz(), [][]int{{0}})
	if len(results) != 1 || results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
}

func TestGradeQuizOrderIndependence(t *testing.T) {
	if results := GradeQuiz(sampleQuiz(), [][]int{{0}, {2, 0}}); !results[1].Passed {
		t.Fatalf("order-independent grading failed: %+v", results)
	}
}

func TestGradeQuizDuplicateIndicesFail(t *testing.T) {
	if results := GradeQuiz(sampleQuiz(), [][]int{{0, 0}, {0, 2}}); results[0].Passed {
		t.Fatalf("duplicate indices must not pass: %+v", results)
	}
}

func TestGradeQuizOutOfRangeFails(t *testing.T) {
	if results := GradeQuiz(sampleQuiz(), [][]int{{9}, {0, 2}}); results[0].Passed {
		t.Fatalf("out-of-range index must not pass: %+v", results)
	}
}
