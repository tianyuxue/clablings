package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInspect(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "inspect.json"))
	if err != nil {
		t.Fatal(err)
	}
	labs, err := ParseInspect(data)
	if err != nil {
		t.Fatalf("ParseInspect: %v", err)
	}
	nodes := labs["clablings-ospf-mtu"]
	if len(nodes) != 1 || nodes[0].Node != "r1" || nodes[0].Container != "clab-clablings-ospf-mtu-r1" {
		t.Fatalf("nodes = %+v", nodes)
	}
	if nodes[0].State != "running" {
		t.Fatalf("state = %q", nodes[0].State)
	}
}

func TestParseInspectInterfaces(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "inspect_interfaces.json"))
	if err != nil {
		t.Fatal(err)
	}
	ifaces, err := ParseInspectInterfaces(data)
	if err != nil {
		t.Fatalf("ParseInspectInterfaces: %v", err)
	}
	list := ifaces["clab-clablings-ospf-mtu-r1"]
	if len(list) != 2 || list[1].Name != "eth1" || list[1].MTU != 1400 || list[1].State != "up" {
		t.Fatalf("ifaces = %+v", list)
	}
}
