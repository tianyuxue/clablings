package content

import "testing"

func TestMissingTranslations(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: Switching, zh: 交换}\nobjectives:\n  en: [a]\n  zh: [甲]\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: quiz\ntitle: {en: t, zh: t}\nhints:\n  - en: only english\nquestions:\n  - prompt: {en: p, zh: p}\n    options:\n      - text: {en: a, zh: a}\n        correct: true\n        explanation: {en: why}\n      - text: {en: b, zh: b}\n",
	})
	warns, err := MissingTranslations(root, "zh")
	if err != nil {
		t.Fatalf("MissingTranslations: %v", err)
	}
	joined := ""
	for _, w := range warns {
		joined += w + "\n"
	}
	for _, want := range []string{"hint 1 missing zh", "option 1 explanation missing zh"} {
		if !contains(joined, want) {
			t.Fatalf("missing %q in:\n%s", want, joined)
		}
	}
	// b has no explanation authored at all -> must not warn for it
	if contains(joined, "option 2 explanation") {
		t.Fatalf("unexpected warning for absent explanation:\n%s", joined)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
