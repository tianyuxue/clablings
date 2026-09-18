package assert

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
)

func TestMatchText(t *testing.T) {
	cases := []struct {
		name string
		got  string
		tm   content.TextMatch
		want bool
	}{
		{"equals-trim", "Full\n", content.TextMatch{Equals: "Full"}, true},
		{"equals-no", "Down", content.TextMatch{Equals: "Full"}, false},
		{"contains", "reply from 10.0.1.1", content.TextMatch{Contains: "reply from"}, true},
		{"regex", "state=Full", content.TextMatch{Regex: `state=(Full|2Way)`}, true},
		{"empty-always", "anything", content.TextMatch{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := matchText(c.got, c.tm)
			if err != nil {
				t.Fatalf("matchText error: %v", err)
			}
			if got != c.want {
				t.Fatalf("matchText = %v, want %v", got, c.want)
			}
		})
	}
}

func TestMatchTextInvalidRegex(t *testing.T) {
	if _, err := matchText("x", content.TextMatch{Regex: "("}); err == nil {
		t.Fatal("expected regex compile error")
	}
}

func TestTokenizePathErrors(t *testing.T) {
	for _, c := range []string{"a", ".", ".a.", "[-1]", "[x]", "["} {
		t.Run(c, func(t *testing.T) {
			if _, err := tokenizePath(c); err == nil {
				t.Fatalf("tokenizePath(%q) expected error", c)
			}
		})
	}
}

func TestEvalJSONPathRequiresDollar(t *testing.T) {
	if _, err := evalJSONPath(map[string]any{}, "neighbors"); err == nil {
		t.Fatal("expected error without leading $")
	}
}

func TestEvalJSONPathNegativeIndex(t *testing.T) {
	if _, err := evalJSONPath(map[string]any{"list": []any{"a"}}, "$.list[-1]"); err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestMatchJSONNoMatchAndInvalidJSON(t *testing.T) {
	if ok, actual, err := matchJSON(`{"a":{}}`, "$.a.*.x", content.TextMatch{}); err != nil || ok || actual != "(JSONPath matched nothing)" {
		t.Fatalf("no-match: ok=%v actual=%q err=%v", ok, actual, err)
	}
	if _, _, err := matchJSON(`not json`, "$.a", content.TextMatch{}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestMatchJSONDistinctValuesDeterministic(t *testing.T) {
	stdout := `{"n":{"b":{"s":"B"},"a":{"s":"A"},"c":{"s":"C"}}}`
	for i := 0; i < 20; i++ {
		_, actual, err := matchJSON(stdout, "$.n.*.s", content.TextMatch{Contains: "A"})
		if err != nil {
			t.Fatal(err)
		}
		if actual != "A, B, C" {
			t.Fatalf("actual = %q, want sorted %q", actual, "A, B, C")
		}
	}
}

func TestEvalJSONPathWildcard(t *testing.T) {
	data := map[string]any{
		"neighbors": map[string]any{
			"10.0.0.1": map[string]any{"state": "Full"},
			"10.0.0.2": map[string]any{"state": "Full"},
		},
	}
	vals, err := evalJSONPath(data, "$.neighbors.*.state")
	if err != nil {
		t.Fatalf("evalJSONPath: %v", err)
	}
	if len(vals) != 2 {
		t.Fatalf("vals = %v", vals)
	}
}

func TestEvalJSONPathIndex(t *testing.T) {
	data := map[string]any{"list": []any{"a", "b", "c"}}
	vals, err := evalJSONPath(data, "$.list[1]")
	if err != nil || len(vals) != 1 || vals[0] != "b" {
		t.Fatalf("vals=%v err=%v", vals, err)
	}
	vals, err = evalJSONPath(data, "$.list[*]")
	if err != nil || len(vals) != 3 {
		t.Fatalf("vals=%v err=%v", vals, err)
	}
}

func TestMatchJSON(t *testing.T) {
	stdout := `{"neighbors":{"10.0.0.1":{"state":"Full"},"10.0.0.2":{"state":"Full"}}}`
	ok, actual, err := matchJSON(stdout, "$.neighbors.*.state", content.TextMatch{Equals: "Full"})
	if err != nil || !ok || actual != "Full, Full" {
		t.Fatalf("ok=%v actual=%q err=%v", ok, actual, err)
	}

	bad := `{"neighbors":{"10.0.0.1":{"state":"ExStart"}}}`
	ok, _, err = matchJSON(bad, "$.neighbors.*.state", content.TextMatch{Equals: "Full"})
	if err != nil || ok {
		t.Fatalf("want failure, ok=%v err=%v", ok, err)
	}
}

func TestMatchJSONInvalidRegexPropagates(t *testing.T) {
	if _, _, err := matchJSON(`{"a":"x"}`, "$.a", content.TextMatch{Regex: "("}); err == nil {
		t.Fatal("expected regex compile error from matchJSON")
	}
}
