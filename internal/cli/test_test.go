package cli

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
)

func TestExitCodeForResults(t *testing.T) {
	if code := exitCodeForResults([]assert.Result{{Passed: true}}); code != 0 {
		t.Fatalf("all passed code = %d", code)
	}
	if code := exitCodeForResults([]assert.Result{{Passed: true}, {Passed: false}}); code != 1 {
		t.Fatalf("failure code = %d", code)
	}
}

func TestParseTestArgs(t *testing.T) {
	if _, err := parseTestArgs(nil, false); err == nil {
		t.Fatal("expected error when no id is given and --all is false")
	}
	id, err := parseTestArgs([]string{"c1/e1"}, false)
	if err != nil || id != "c1/e1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := parseTestArgs(nil, true); err != nil {
		t.Fatalf("--all without id: %v", err)
	}
	if _, err := parseTestArgs([]string{"c1/e1"}, true); err == nil {
		t.Fatal("expected error combining --all with an id")
	}
}

func TestUnique(t *testing.T) {
	got := unique([]string{"a", "", "b", "a", ""})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unique = %v", got)
	}
}
