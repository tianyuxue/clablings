package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/assert"
)

func TestTextFormat(t *testing.T) {
	results := []assert.Result{
		{Name: "OSPF neighbor reaches Full", Type: "exec", Layer: "control-plane", Passed: true},
		{
			Name: "PBR forwarding", Type: "exec", Layer: "data-plane", Passed: false,
			Expected: "packet egress via eth1", Actual: "packet egress via eth2", Hint: "check ip rule",
		},
	}
	var b bytes.Buffer
	Text(&b, results)
	out := b.String()
	if !strings.Contains(out, "✓ [control-plane] OSPF neighbor reaches Full") {
		t.Fatalf("missing pass line:\n%s", out)
	}
	if !strings.Contains(out, "✗ [data-plane] PBR forwarding") {
		t.Fatalf("missing fail line:\n%s", out)
	}
	for _, want := range []string{"Expected: packet egress via eth1", "Actual:   packet egress via eth2", "Hint:     check ip rule"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
}

func TestJSONFormat(t *testing.T) {
	results := []assert.Result{{Name: "a", Type: "exec", Passed: true}}
	var b bytes.Buffer
	if err := JSON(&b, results); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var back []map[string]any
	if err := json.Unmarshal(b.Bytes(), &back); err != nil {
		t.Fatalf("output is not JSON: %v", err)
	}
	if len(back) != 1 || back[0]["name"] != "a" || back[0]["passed"] != true {
		t.Fatalf("json = %v", back)
	}
}

func TestAllPassed(t *testing.T) {
	if !AllPassed(nil) {
		t.Fatal("empty should pass")
	}
	if !AllPassed([]assert.Result{{Passed: true}, {Passed: true}}) {
		t.Fatal("all pass expected")
	}
	if AllPassed([]assert.Result{{Passed: true}, {Passed: false}}) {
		t.Fatal("mixed should fail")
	}
	if AllPassed([]assert.Result{{Passed: false}}) {
		t.Fatal("all fail should fail")
	}
}

func TestTextErrorBranch(t *testing.T) {
	results := []assert.Result{{Name: "boom", Type: "exec", Passed: false, Err: errors.New("kaboom")}}
	var b bytes.Buffer
	Text(&b, results)
	out := b.String()
	if !strings.Contains(out, "✗ boom") || !strings.Contains(out, "Error:    kaboom") {
		t.Fatalf("out = %q", out)
	}
	if strings.Contains(out, "Expected:") || strings.Contains(out, "Actual:") || strings.Contains(out, "Hint:") {
		t.Fatalf("empty optional fields should be omitted: %q", out)
	}
}

func TestJSONOptionalFieldsAndError(t *testing.T) {
	results := []assert.Result{{Name: "a", Type: "exec", Passed: false, Layer: "data-plane", Viewpoint: "r1", Expected: "e", Actual: "a", Hint: "h", Err: errors.New("kaboom")}}
	var b bytes.Buffer
	if err := JSON(&b, results); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var back []map[string]any
	if err := json.Unmarshal(b.Bytes(), &back); err != nil {
		t.Fatal(err)
	}
	got := back[0]
	for k, want := range map[string]any{"name": "a", "layer": "data-plane", "viewpoint": "r1", "expected": "e", "actual": "a", "hint": "h", "error": "kaboom", "passed": false} {
		if got[k] != want {
			t.Fatalf("json[%q] = %v, want %v (%v)", k, got[k], want, got)
		}
	}
}

func TestJSONEmptyResultsIsArray(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(b.String()) != "[]" {
		t.Fatalf("empty JSON = %q, want []", b.String())
	}
}
