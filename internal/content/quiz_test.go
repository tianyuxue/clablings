package content

import (
	"strings"
	"testing"
)

func TestLoadQuiz(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: Concepts\n",
		"chapters/c1/exercises/q1/exercise.yaml": "id: q1\ntype: quiz\ntitle: Why STP\ntimeout: 30s\nquestions:\n  - prompt: {en: \"What breaks first?\", zh: 先坏什么？}\n    multiple: false\n    options:\n      - text: {en: Storm, zh: 风暴}\n        correct: true\n        explanation: {en: loops flood, zh: 环路泛洪}\n      - text: {en: Nothing, zh: 没事}\n        correct: false\n        explanation: {en: nope, zh: 不对}\n",
	})
	lib, err := LoadWithLocale(root, "zh")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ex := lib.Chapters[0].Exercises[0]
	if ex.Type != TypeQuiz || len(ex.Questions) != 1 {
		t.Fatalf("exercise = %+v", ex)
	}
	q := ex.Questions[0]
	if q.Prompt != "先坏什么？" || len(q.Options) != 2 || !q.Options[0].Correct || q.Options[0].Explanation != "环路泛洪" {
		t.Fatalf("question = %+v", q)
	}
}

func TestQuizValidationErrors(t *testing.T) {
	base := func(exerciseYAML string) map[string]string {
		return map[string]string{
			"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
			"chapters/c1/chapter.yaml":               "id: c1\ntitle: t\n",
			"chapters/c1/exercises/q1/exercise.yaml": exerciseYAML,
		}
	}
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "no questions",
			body: "id: q1\ntype: quiz\ntitle: t\nquestions: []\n",
			want: "at least one question",
		},
		{
			name: "too few options",
			body: "id: q1\ntype: quiz\ntitle: t\nquestions:\n  - prompt: p\n    options:\n      - text: a\n        correct: true\n",
			want: "at least two options",
		},
		{
			name: "no correct option",
			body: "id: q1\ntype: quiz\ntitle: t\nquestions:\n  - prompt: p\n    options:\n      - text: a\n      - text: b\n",
			want: "correct option",
		},
		{
			name: "single choice with two correct",
			body: "id: q1\ntype: quiz\ntitle: t\nquestions:\n  - prompt: p\n    multiple: false\n    options:\n      - text: a\n        correct: true\n      - text: b\n        correct: true\n",
			want: "exactly one correct",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Load(writeLib(t, base(tc.body)))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want contains %q", err, tc.want)
			}
		})
	}
}
