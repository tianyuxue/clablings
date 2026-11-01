package content

import (
	"strings"
	"testing"
)

func TestTranslationParityConceptDrift(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n\n## Terms\n\n```sh\nbridge vlan add dev eth1 vid 10\n```\n",
		"chapters/c1/concept.zh.md": "# 标题\n\n## 术语\n\n```sh\nbridge vlan add dev eth1 vid 20\n```\n",
	})
	problems, err := TranslationParity(root, "zh")
	if err != nil {
		t.Fatalf("TranslationParity: %v", err)
	}
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "code block") {
		t.Fatalf("expected code block drift, got:\n%s", joined)
	}
}

func TestTranslationParityAllowsDifferentStructure(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n\n## A\n\nShort.\n\n## B\n\nMore.\n\n```sh\nbridge vlan show\n```\n",
		"chapters/c1/concept.zh.md": "# 标题\n\n## 术语\n\n中文可以重新分段，并使用不同的行内代码。\n\n```sh\nbridge vlan show\n```\n",
	})
	problems, err := TranslationParity(root, "zh")
	if err != nil {
		t.Fatalf("TranslationParity: %v", err)
	}
	if len(problems) != 0 {
		t.Fatalf("structure may differ, got %v", problems)
	}
}

func TestTranslationParityMissingConcept(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":            "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":  "id: c1\ntitle: {en: Switching, zh: 交换}\n",
		"chapters/c1/concept.en.md": "# T\n",
	})
	problems, err := TranslationParity(root, "zh")
	if err != nil {
		t.Fatalf("TranslationParity: %v", err)
	}
	if !strings.Contains(strings.Join(problems, "\n"), "concept.zh.md missing") {
		t.Fatalf("expected missing concept problem, got %v", problems)
	}
}
