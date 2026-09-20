package content

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/i18n"
)

func fakeLast(times map[string]time.Time) i18n.LastCommit {
	return func(p string) (time.Time, bool, error) {
		t, ok := times[p]
		return t, ok, nil
	}
}

func TestTranslationParityConceptDrift(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n\n## Terms\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n",
		"chapters/c1/concept.zh.md": "# 标题\n\n## 术语\n\n```sh\nbridge vlan add dev eth1 vid 20\n```\n",
	})
	problems, _, err := translationParity(root, "zh", fakeLast(nil))
	if err != nil {
		t.Fatalf("translationParity: %v", err)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "code block 1") {
		t.Fatalf("expected code block drift, got:\n%s", joined)
	}
}

func TestTranslationParityMissingConcept(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n",
	})
	problems, _, err := translationParity(root, "zh", fakeLast(nil))
	if err != nil {
		t.Fatalf("translationParity: %v", err)
	}
	if !strings.Contains(strings.Join(problems, "\n"), "concept.zh.md missing") {
		t.Fatalf("expected missing concept problem, got %v", problems)
	}
}

func TestTranslationParityStaleWarns(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n",
		"chapters/c1/concept.zh.md": "# 标题\n",
	})
	en := filepath.Join(root, "chapters/c1/concept.en.md")
	zh := filepath.Join(root, "chapters/c1/concept.zh.md")
	last := fakeLast(map[string]time.Time{
		en: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		zh: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	problems, warns, err := translationParity(root, "zh", last)
	if err != nil {
		t.Fatalf("translationParity: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("problems = %v, want none", problems)
	}
	if !strings.Contains(strings.Join(warns, "\n"), "may be outdated") {
		t.Fatalf("expected staleness warning, got %v", warns)
	}
}
