package assert

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func TestRunFilteredKeep(t *testing.T) {
	ex := &content.Exercise{ID: "01", Chapter: "01", Assertions: []content.Assertion{
		{Name: "a", Type: "exec", Node: "r1", Cmd: "true"},
		{Name: "b", Type: "exec", Node: "r1", Cmd: "true"},
	}}
	fe := &fakeExec{out: map[string]engine.Result{"true": {Code: 0}}}
	if all := RunFiltered(context.Background(), fe, nil, ex, nil); len(all) != 2 {
		t.Fatalf("nil keep ran %d assertions, want 2", len(all))
	}
	onlyA := RunFiltered(context.Background(), fe, nil, ex, func(a content.Assertion) bool { return a.Name == "a" })
	if len(onlyA) != 1 || onlyA[0].Name != "a" {
		t.Fatalf("keep filter = %+v", onlyA)
	}
}

func TestRunExerciseInterfaceInspectShared(t *testing.T) {
	ex := &content.Exercise{
		ID: "01", Chapter: "01",
		ChapterRef: &content.Chapter{TopologyName: "lab"},
		Assertions: []content.Assertion{
			{Name: "iface1", Type: "interface", Node: "r1", Interface: "eth1", State: "up"},
			{Name: "iface2", Type: "interface", Node: "r2", Interface: "eth1", State: "up"},
		},
	}
	fe := &fakeExec{iface: map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
		"clab-lab-r2": {{Name: "eth1", State: "up"}},
	}}
	results := RunExercise(context.Background(), fe, nil, ex)
	if len(results) != 2 || !results[0].Passed || !results[1].Passed {
		t.Fatalf("results = %+v", results)
	}
	if fe.ifaceCalls != 1 {
		t.Fatalf("InspectInterfaces calls = %d, want 1", fe.ifaceCalls)
	}
}

func TestRunExerciseInspectError(t *testing.T) {
	ex := &content.Exercise{
		ID: "01", Chapter: "01",
		ChapterRef: &content.Chapter{TopologyName: "lab"},
		Assertions: []content.Assertion{{Name: "iface", Type: "interface", Node: "r1", Interface: "eth1", State: "up"}},
	}
	fe := &fakeExec{ifaceErr: errors.New("inspect boom")}
	results := RunExercise(context.Background(), fe, nil, ex)
	if len(results) != 1 || results[0].Passed || results[0].Err == nil {
		t.Fatalf("results = %+v", results)
	}
}

func TestRunExerciseDispatchesFileAndLink(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), []byte("clean"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &content.Exercise{
		ID: "01", Chapter: "01", Dir: dir,
		ChapterRef: &content.Chapter{TopologyName: "lab"},
		Assertions: []content.Assertion{
			{Name: "file", Type: "file", Source: "ok.txt"},
			{Name: "link", Type: "link", A: "r1", PortA: "eth1", B: "r2", PortB: "eth1", State: "up"},
		},
	}
	topo := &engine.Topology{Links: []engine.TopoLink{{A: "r1", PortA: "eth1", B: "r2", PortB: "eth1"}}}
	fe := &fakeExec{iface: map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
		"clab-lab-r2": {{Name: "eth1", State: "up"}},
	}}
	results := RunExercise(context.Background(), fe, topo, ex)
	if len(results) != 2 || !results[0].Passed || !results[1].Passed {
		t.Fatalf("results = %+v", results)
	}
}
