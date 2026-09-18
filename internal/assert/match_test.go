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
			if got := matchText(c.got, c.tm); got != c.want {
				t.Fatalf("matchText = %v, want %v", got, c.want)
			}
		})
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
