package assert

import (
	"strings"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func runInterface(ex *content.Exercise, a *content.Assertion, ifaces map[string][]engine.InspectInterface) Result {
	container := containerName(ex, a.Node)
	state := interfaceState(ifaces, container, a.Interface)
	expected := a.Node + ":" + a.Interface + " state=" + a.State
	if state == "" {
		return Result{Passed: false, Expected: expected, Actual: "interface not found"}
	}
	return Result{Passed: state == a.State, Expected: expected, Actual: "state=" + state}
}

func runLink(ex *content.Exercise, a *content.Assertion, topo *engine.Topology, ifaces map[string][]engine.InspectInterface) Result {
	if topo != nil && !hasLink(topo, a) {
		return Result{Passed: false, Expected: linkDesc(a, a.State), Actual: "link not present in topology"}
	}
	sa := interfaceState(ifaces, containerName(ex, a.A), a.PortA)
	sb := interfaceState(ifaces, containerName(ex, a.B), a.PortB)

	actualState := "down"
	switch {
	case sa == "" || sb == "":
		actualState = "unknown"
	case sa == "up" && sb == "up":
		actualState = "up"
	}
	passed := actualState == a.State
	if a.State == "unknown" {
		passed = actualState == "unknown"
	}
	return Result{Passed: passed, Expected: linkDesc(a, a.State), Actual: linkDesc(a, actualState)}
}

func linkDesc(a *content.Assertion, state string) string {
	return a.A + ":" + a.PortA + " <-> " + a.B + ":" + a.PortB + " " + state
}

func hasLink(topo *engine.Topology, a *content.Assertion) bool {
	for _, l := range topo.Links {
		if l.A == a.A && l.PortA == a.PortA && l.B == a.B && l.PortB == a.PortB {
			return true
		}
		if l.A == a.B && l.PortA == a.PortB && l.B == a.A && l.PortB == a.PortA {
			return true
		}
	}
	return false
}

func interfaceState(ifaces map[string][]engine.InspectInterface, container, name string) string {
	for _, i := range ifaces[container] {
		if strings.EqualFold(i.Name, name) {
			return i.State
		}
	}
	return ""
}
