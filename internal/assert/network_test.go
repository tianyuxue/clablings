package assert

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestInterfaceAssertion(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	ifaces := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
	}
	a := &content.Assertion{Type: "interface", Node: "r1", Interface: "eth1", State: "up"}
	res := runInterface(ex, a, ifaces)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}
	a.State = "down"
	res = runInterface(ex, a, ifaces)
	if res.Passed {
		t.Fatalf("want failure: %+v", res)
	}
}

func TestLinkAssertion(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	topo := &engine.Topology{Links: []engine.TopoLink{{A: "r1", PortA: "eth1", B: "r2", PortB: "eth1"}}}
	ifaces := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
		"clab-lab-r2": {{Name: "eth1", State: "up"}},
	}
	a := &content.Assertion{Type: "link", A: "r1", PortA: "eth1", B: "r2", PortB: "eth1", State: "up"}
	res := runLink(ex, a, topo, ifaces)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}

	ifaces["clab-lab-r2"] = []engine.InspectInterface{{Name: "eth1", State: "down"}}
	res = runLink(ex, a, topo, ifaces)
	if res.Passed {
		t.Fatalf("want failure: %+v", res)
	}
}

func TestRunInterfaceBranches(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	ifaces := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}, {Name: "eth2", State: "down"}},
	}
	cases := []struct {
		name       string
		a          content.Assertion
		wantPassed bool
	}{
		{"up pass", content.Assertion{Node: "r1", Interface: "eth1", State: "up"}, true},
		{"down pass", content.Assertion{Node: "r1", Interface: "eth2", State: "down"}, true},
		{"state mismatch", content.Assertion{Node: "r1", Interface: "eth1", State: "down"}, false},
		{"interface missing", content.Assertion{Node: "r1", Interface: "eth9", State: "up"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runInterface(ex, &c.a, ifaces)
			if got.Passed != c.wantPassed {
				t.Fatalf("passed = %v, want %v (%+v)", got.Passed, c.wantPassed, got)
			}
		})
	}
}

func TestRunLinkBranches(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	topo := &engine.Topology{Links: []engine.TopoLink{{A: "r1", PortA: "eth1", B: "r2", PortB: "eth1"}}}
	bothUp := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
		"clab-lab-r2": {{Name: "eth1", State: "up"}},
	}
	oneDown := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
		"clab-lab-r2": {{Name: "eth1", State: "down"}},
	}
	missing := map[string][]engine.InspectInterface{
		"clab-lab-r1": {{Name: "eth1", State: "up"}},
	}
	node := content.Assertion{A: "r1", PortA: "eth1", B: "r2", PortB: "eth1"}
	mk := func(state string) content.Assertion { a := node; a.State = state; return a }

	cases := []struct {
		name       string
		a          content.Assertion
		topo       *engine.Topology
		ifaces     map[string][]engine.InspectInterface
		wantPassed bool
	}{
		{"both up expects up", mk("up"), topo, bothUp, true},
		{"one down expects up", mk("up"), topo, oneDown, false},
		{"one down expects down", mk("down"), topo, oneDown, true},
		{"missing expects unknown", mk("unknown"), topo, missing, true},
		{"missing expects up", mk("up"), topo, missing, false},
		{"reversed endpoint order", content.Assertion{A: "r2", PortA: "eth1", B: "r1", PortB: "eth1", State: "up"}, topo, bothUp, true},
		{"link not in topology", content.Assertion{A: "r1", PortA: "eth9", B: "r2", PortB: "eth9", State: "up"}, topo, bothUp, false},
		{"nil topology skips presence check", mk("up"), nil, bothUp, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := runLink(ex, &c.a, c.topo, c.ifaces)
			if got.Passed != c.wantPassed {
				t.Fatalf("passed = %v, want %v (%+v)", got.Passed, c.wantPassed, got)
			}
		})
	}
}
