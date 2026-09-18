package assert

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestRunExerciseExecPass(t *testing.T) {
	fe := &fakeExec{out: map[string]engine.Result{
		`vtysh -c "show ip ospf neighbor json"`: {Stdout: `{"neighbors":{"10.0.0.1":{"state":"Full"}}}`, Code: 0},
	}}
	ex := &content.Exercise{
		ID: "01", Chapter: "01",
		ChapterRef: &content.Chapter{TopologyName: "clablings-ospf-mtu"},
		Assertions: []content.Assertion{{
			Name: "OSPF Full", Type: "exec", Layer: "control-plane", Node: "r1",
			Cmd:   `vtysh -c "show ip ospf neighbor json"`,
			Match: &content.Match{JSON: "$.neighbors.*.state", TextMatch: content.TextMatch{Equals: "Full"}},
		}},
	}
	results := RunExercise(context.Background(), fe, nil, ex)
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("results = %+v", results)
	}
	if results[0].Exercise != "01/01" || results[0].Layer != "control-plane" {
		t.Fatalf("result metadata = %+v", results[0])
	}
	if fe.execs[0] != `clab-clablings-ospf-mtu-r1: vtysh -c "show ip ospf neighbor json"` {
		t.Fatalf("exec = %q", fe.execs[0])
	}
}

func TestRunExerciseExecFail(t *testing.T) {
	fe := &fakeExec{out: map[string]engine.Result{"echo hi": {Stdout: "bye\n", Code: 0}}}
	ex := &content.Exercise{
		ID: "01", Chapter: "01",
		ChapterRef: &content.Chapter{TopologyName: "lab"},
		Assertions: []content.Assertion{{
			Name: "echo", Type: "exec", Node: "r1", Cmd: "echo hi",
			Match: &content.Match{TextMatch: content.TextMatch{Contains: "hi"}},
		}},
	}
	results := RunExercise(context.Background(), fe, nil, ex)
	if results[0].Passed {
		t.Fatalf("want failure: %+v", results[0])
	}
	if results[0].Actual == "" || results[0].Expected == "" {
		t.Fatalf("Expected/Actual must not be empty: %+v", results[0])
	}
}

func TestRunExerciseUnknownTypeFails(t *testing.T) {
	fe := &fakeExec{}
	ex := &content.Exercise{ID: "01", Chapter: "01", Assertions: []content.Assertion{{Name: "x", Type: "bogus"}}}
	results := RunExercise(context.Background(), fe, nil, ex)
	if results[0].Passed || results[0].Actual == "" {
		t.Fatalf("want failure with Actual: %+v", results[0])
	}
}
