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
