package report

import (
	"bytes"
	"encoding/json"
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
