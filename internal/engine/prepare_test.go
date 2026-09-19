package engine

import (
	"os"
	"path/filepath"
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
