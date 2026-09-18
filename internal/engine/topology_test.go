package engine

import (
	"os"
	"path/filepath"
	"strings"
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

func TestParseTopologyErrors(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{"missing name", "topology:\n  nodes: {}\n", "missing top-level name"},
		{"bad endpoint count", "name: l\ntopology:\n  links:\n    - endpoints: [\"r1:eth1\"]\n", "exactly 2 endpoints"},
		{"invalid endpoint", "name: l\ntopology:\n  links:\n    - endpoints: [\"r1\", \"r2:eth1\"]\n", "invalid endpoint"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".clab.yml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := ParseTopology(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want contains %q", err, tc.want)
			}
		})
	}
}

func TestParseTopologySortsNodes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "t.clab.yml")
	if err := os.WriteFile(path, []byte("name: l\ntopology:\n  nodes:\n    z: {kind: linux}\n    a: {kind: linux}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	topo, err := ParseTopology(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(topo.Nodes) != 2 || topo.Nodes[0].Name != "a" || topo.Nodes[1].Name != "z" {
		t.Fatalf("nodes = %+v", topo.Nodes)
	}
}
