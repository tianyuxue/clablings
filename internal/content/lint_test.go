package content

import (
	"strings"
	"testing"
)

func TestLegacyLocalizedFiles(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: 交换\n",
		"chapters/c1/concept.md":                 "# 交换\n",
		"chapters/c1/reading.yaml":               "- title: RFC\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: quiz\ntitle: t\nbrief: b\nquestions:\n  - prompt: p\n    options: [{text: a, correct: true}, {text: b}]\n",
		"chapters/c1/exercises/e1/README.md":     "# 题\n",
		"chapters/c1/exercises/e2/exercise.yaml": "id: e2\ntype: quiz\ntitle: t\nbrief: b\nquestions:\n  - prompt: p\n    options: [{text: a, correct: true}, {text: b}]\n",
		"chapters/c1/exercises/e2/README.en.md":  "# Quiz\n",
		"chapters/c1/exercises/e2/README.zh.md":  "# 题目\n",
		"chapters/c1/concept.en.md":              "# Switching\n",
		"chapters/c1/concept.zh.md":              "# 交换\n",
		"chapters/c1/reading.en.yaml":            "- title: RFC\n",
	})
	problems, err := LegacyLocalizedFiles(root)
	if err != nil {
		t.Fatalf("LegacyLocalizedFiles: %v", err)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"concept.en.md", "concept.zh.md", "README.en.md", "README.zh.md", "reading.en.yaml"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	for _, clean := range []string{"concept.md", "README.md", "reading.yaml"} {
		if strings.Contains(joined, clean+":") {
			t.Fatalf("false positive for %q:\n%s", clean, joined)
		}
	}
}
