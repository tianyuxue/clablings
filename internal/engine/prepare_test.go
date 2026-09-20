package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
)

func TestPrepareTopologyRewritesName(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "topology.clab.yml")
	if err := os.WriteFile(orig, []byte("name: static\ntopology:\n  nodes:\n    r1:\n      kind: linux\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := &content.Chapter{ID: "c1", TopologyPath: orig, TopologyName: "static", LabName: "inst-static"}
	got, err := PrepareTopology(ch)
	if err != nil {
		t.Fatalf("PrepareTopology: %v", err)
	}
	if got != DerivedTopologyPath(ch) {
		t.Fatalf("path = %q, want %q", got, DerivedTopologyPath(ch))
	}
	topo, err := ParseTopology(got)
	if err != nil {
		t.Fatalf("ParseTopology: %v", err)
	}
	if topo.Name != "inst-static" {
		t.Fatalf("name = %q, want inst-static", topo.Name)
	}
	if topo.Nodes[0].Name != "r1" {
		t.Fatalf("nodes = %+v", topo.Nodes)
	}
}

func TestPrepareTopologyInsertsNameWhenMissing(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "topology.clab.yml")
	if err := os.WriteFile(orig, []byte("topology:\n  nodes: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := &content.Chapter{TopologyPath: orig, LabName: "x-y"}
	if _, err := PrepareTopology(ch); err != nil {
		t.Fatalf("PrepareTopology: %v", err)
	}
	topo, err := ParseTopology(DerivedTopologyPath(ch))
	if err != nil {
		t.Fatalf("ParseTopology: %v", err)
	}
	if topo.Name != "x-y" {
		t.Fatalf("name = %q", topo.Name)
	}
}

func TestPrepareTopologyEmptyLabNameUsesOriginal(t *testing.T) {
	ch := &content.Chapter{TopologyPath: "/does/not/matter.yml"}
	got, err := PrepareTopology(ch)
	if err != nil || got != ch.TopologyPath {
		t.Fatalf("got=%q err=%v", got, err)
	}
}

func TestDerivedTopologyPathIsPerInstance(t *testing.T) {
	a := &content.Chapter{TopologyPath: "/x/topology.clab.yml", LabName: "inst-a"}
	b := &content.Chapter{TopologyPath: "/x/topology.clab.yml", LabName: "inst-b"}
	if DerivedTopologyPath(a) == DerivedTopologyPath(b) {
		t.Fatalf("paths must differ: %q", DerivedTopologyPath(a))
	}
}

func TestPrepareTopologyPreservesNodeGroup(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "topology.clab.yml")
	src := "name: static\ntopology:\n  nodes:\n    sw:\n      kind: linux\n      group: switch\n    h1:\n      kind: linux\n      group: host\n"
	if err := os.WriteFile(orig, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := &content.Chapter{ID: "c1", TopologyPath: orig, TopologyName: "static", LabName: "inst-static"}
	got, err := PrepareTopology(ch)
	if err != nil {
		t.Fatalf("PrepareTopology: %v", err)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"group: switch", "group: host"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("derived topology dropped %q:\n%s", want, data)
		}
	}
}

func TestPrepareTopologyRejectsNonMapping(t *testing.T) {
	dir := t.TempDir()
	orig := filepath.Join(dir, "topology.clab.yml")
	if err := os.WriteFile(orig, []byte("- a\n- b\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := &content.Chapter{TopologyPath: orig, LabName: "x"}
	if _, err := PrepareTopology(ch); err == nil {
		t.Fatal("expected error for non-mapping topology")
	}
}
