package engine

import (
	"path/filepath"
	"testing"
)

func TestParseTopology(t *testing.T) {
	topo, err := ParseTopology(filepath.Join("testdata", "topo.clab.yml"))
	if err != nil {
		t.Fatalf("ParseTopology: %v", err)
	}
	if topo.Name != "clablings-ospf-mtu" {
		t.Fatalf("name = %q", topo.Name)
	}
	if len(topo.Nodes) != 2 || topo.Nodes[0].Name != "r1" || topo.Nodes[0].Kind != "linux" {
		t.Fatalf("nodes = %+v", topo.Nodes)
	}
	if len(topo.Links) != 1 {
		t.Fatalf("links = %+v", topo.Links)
	}
	l := topo.Links[0]
	if l.A != "r1" || l.PortA != "eth1" || l.B != "r2" || l.PortB != "eth1" {
		t.Fatalf("link = %+v", l)
	}
}

func TestContainerName(t *testing.T) {
	if got := ContainerName("clablings-ospf-mtu", "r1"); got != "clab-clablings-ospf-mtu-r1" {
		t.Fatalf("ContainerName = %q", got)
	}
}
