package content

import "testing"

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
