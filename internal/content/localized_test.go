package content

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"": "en", "  ": "en", "en": "en", "EN": "en", "en-US": "en",
		"zh": "zh", "zh-CN": "zh", "zh-Hans": "zh", "fr": "fr",
		"-en-US": "en", "_zh": "zh", ".zh": "zh",
	}
	for in, want := range cases {
		if got := NormalizeLocale(in); got != want {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalizedStringDecodeAndResolve(t *testing.T) {
	var l localizedString
	if err := yaml.Unmarshal([]byte("zh-Hans: 你好\nen: hi\n"), &l); err != nil {
		t.Fatal(err)
	}
	if got := l.resolve("zh"); got != "你好" {
		t.Fatalf("zh = %q", got)
	}
	if got := l.resolve("en"); got != "hi" {
		t.Fatalf("en = %q", got)
	}
	if got := l.resolve("fr"); got != "hi" {
		t.Fatalf("fr fallback = %q", got)
	}
}

func TestLocalizedStringScalar(t *testing.T) {
	var l localizedString
	if err := yaml.Unmarshal([]byte("plain"), &l); err != nil {
		t.Fatal(err)
	}
	if got := l.resolve("zh"); got != "plain" {
		t.Fatalf("scalar = %q", got)
	}
}

func TestLocalizedStringSequenceErrors(t *testing.T) {
	var l localizedString
	if err := yaml.Unmarshal([]byte("- a\n"), &l); err == nil {
		t.Fatal("expected error for sequence")
	}
}

func TestResolveStringsAndList(t *testing.T) {
	ss := resolveStrings([]localizedString{
		{Values: map[string]string{"zh": "甲"}},
		{Values: map[string]string{"en": "b"}},
	}, "zh")
	if len(ss) != 2 || ss[0] != "甲" || ss[1] != "b" {
		t.Fatalf("resolveStrings = %v", ss)
	}
	list := resolveLocalizedList(map[string][]string{"zh-Hans": {"甲"}, "en": {"a"}}, "zh")
	if len(list) != 1 || list[0] != "甲" {
		t.Fatalf("resolveLocalizedList = %v", list)
	}
	if got := resolveLocalizedList(map[string][]string{"fr": {"x"}}, "zh"); len(got) != 1 || got[0] != "x" {
		t.Fatalf("any-available fallback = %v, want [x]", got)
	}
	// duplicate normalized keys: deterministic (sorted key order picks "zh")
	dup := map[string][]string{"zh-Hans": {"hans"}, "zh": {"plain"}}
	if got := resolveLocalizedList(dup, "zh"); len(got) != 1 || got[0] != "plain" {
		t.Fatalf("duplicate normalized keys = %v, want [plain]", got)
	}
	// default key normalization: en-US alone must satisfy the en fallback
	enUS := map[string][]string{"en-US": {"hello"}}
	if got := resolveLocalizedList(enUS, "zh"); len(got) != 1 || got[0] != "hello" {
		t.Fatalf("en-US fallback = %v", got)
	}
}

func TestResolveAnyAvailableFallback(t *testing.T) {
	if got := (localizedString{Values: map[string]string{"fr": "bonjour"}}).resolve("de"); got != "bonjour" {
		t.Fatalf("resolve = %q", got)
	}
	var l localizedString
	if err := yaml.Unmarshal([]byte("zh: 甲\nzh-CN: 乙\n"), &l); err != nil {
		t.Fatal(err)
	}
	if got := l.resolve("zh"); got != "乙" { // document order: later duplicate wins
		t.Fatalf("duplicate normalized keys = %q, want 乙", got)
	}
}

func writeLib(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLoadWithLocaleResolvesAndFallsBack(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle:\n  en: Switching\n  zh: 交换\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes: {}\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: config-fix\ntitle: Plain title\nhints:\n  - en: an english hint\n    zh: 一条中文线索\n  - scalar hint\nassertions:\n  - name:\n      en: neighbor up\n      zh: 邻居 up\n    type: exec\n    node: n1\n    cmd: \"true\"\n    hint: only english\n",
	})

	lib, err := LoadWithLocale(root, "zh")
	if err != nil {
		t.Fatalf("LoadWithLocale zh: %v", err)
	}
	ch := lib.Chapters[0]
	ex := ch.Exercises[0]
	if ch.Title != "交换" {
		t.Fatalf("chapter title = %q", ch.Title)
	}
	if ex.Title != "Plain title" {
		t.Fatalf("exercise title = %q", ex.Title)
	}
	if len(ex.Hints) != 2 || ex.Hints[0] != "一条中文线索" || ex.Hints[1] != "scalar hint" {
		t.Fatalf("hints = %v", ex.Hints)
	}
	if ex.Assertions[0].Name != "邻居 up" || ex.Assertions[0].Hint != "only english" {
		t.Fatalf("assertion = %+v", ex.Assertions[0])
	}

	enLib, err := LoadWithLocale(root, "en")
	if err != nil {
		t.Fatalf("LoadWithLocale en: %v", err)
	}
	if enLib.Chapters[0].Title != "Switching" || enLib.Chapters[0].Exercises[0].Hints[0] != "an english hint" {
		t.Fatalf("en resolution wrong")
	}
}

func TestLoadExerciseNilAssertions(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: t\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes: {}\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: config-fix\n",
	})
	lib, err := LoadWithLocale(root, "en")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := lib.Chapters[0].Exercises[0].Assertions; got != nil {
		t.Fatalf("assertions = %v, want nil", got)
	}
}

func TestLoadWithLocaleMetadata(t *testing.T) {
	root := writeLib(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: STP\ndifficulty: 3\ntags: [l2, stp]\nestimated_minutes: 30\nprerequisites: [02-vlan]\nobjectives:\n  en: [a, b]\n  zh: [甲, 乙]\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes: {}\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntype: config-fix\ndifficulty: 2\ntags: [stp]\nassertions:\n  - name: x\n    type: exec\n    node: n1\n    cmd: \"true\"\n",
	})
	lib, err := LoadWithLocale(root, "zh")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ch := lib.Chapters[0]
	if ch.Difficulty != 3 || len(ch.Tags) != 2 || ch.Tags[0] != "l2" || ch.EstimatedMinutes != 30 || len(ch.Prerequisites) != 1 || ch.Prerequisites[0] != "02-vlan" {
		t.Fatalf("metadata = %+v", ch)
	}
	if len(ch.Objectives) != 2 || ch.Objectives[0] != "甲" {
		t.Fatalf("objectives = %v", ch.Objectives)
	}
	ex := lib.Chapters[0].Exercises[0]
	if ex.Difficulty != 2 || len(ex.Tags) != 1 || ex.Tags[0] != "stp" {
		t.Fatalf("exercise metadata = %+v", ex)
	}
}
