package content

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"": "en", "  ": "en", "en": "en", "EN": "en", "en-US": "en",
		"zh": "zh", "zh-CN": "zh", "zh-Hans": "zh", "fr": "fr",
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
	if resolveLocalizedList(map[string][]string{"fr": {"x"}}, "zh") != nil {
		t.Fatal("expected nil when no match and no default")
	}
}
