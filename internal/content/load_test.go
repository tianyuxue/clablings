package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLibrary(t *testing.T) {
	root := filepath.Join("testdata", "library")
	lib, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if lib.Manifest.Version != 1 {
		t.Fatalf("version = %d", lib.Manifest.Version)
	}
	if len(lib.Chapters) != 2 {
		t.Fatalf("chapters = %d, 期望 2", len(lib.Chapters))
	}
	if len(lib.Order) != 2 {
		t.Fatalf("order = %d, 期望 2", len(lib.Order))
	}

	c0 := lib.Chapters[0]
	if c0.ID != "01-ospf-mtu" || c0.Title != "OSPF MTU 不匹配" {
		t.Fatalf("chapter0 = %+v", c0)
	}
	if c0.TopologyPath != filepath.Join(root, "chapters", "01-ospf-mtu", "topology.clab.yml") {
		t.Fatalf("topology path = %q", c0.TopologyPath)
	}
	if c0.TopologyName != "clablings-ospf-mtu" {
		t.Fatalf("topology name = %q", c0.TopologyName)
	}
	if len(c0.Exercises) != 1 {
		t.Fatalf("exercises = %d", len(c0.Exercises))
	}

	ex := lib.ByID["01-ospf-mtu/01_mtu"]
	if ex == nil {
		t.Fatal("ByID 缺少 01-ospf-mtu/01_mtu")
	}
	if ex.Type != TypeConfigFix {
		t.Fatalf("type = %q", ex.Type)
	}
	if len(ex.Files) != 1 || ex.Files[0].Source != "frr1/frr.conf" || ex.Files[0].Node != "r1" {
		t.Fatalf("files = %+v", ex.Files)
	}
	if len(ex.Assertions) != 1 || ex.Assertions[0].Type != "exec" {
		t.Fatalf("assertions = %+v", ex.Assertions)
	}
	if ex.Assertions[0].Match == nil || ex.Assertions[0].Match.JSON != "$.neighbors.*.state" {
		t.Fatalf("match = %+v", ex.Assertions[0].Match)
	}
	if ex.Dir != filepath.Join(root, "chapters", "01-ospf-mtu", "exercises", "01_mtu") {
		t.Fatalf("exercise dir = %q", ex.Dir)
	}

	code := lib.ByID["02-arp/01_arp"]
	if code == nil || code.Build == nil || code.Run == nil {
		t.Fatalf("code exercise = %+v", code)
	}
	if code.Build.Image != "gcc:13" || code.Build.Output != "arp_responder" {
		t.Fatalf("build = %+v", code.Build)
	}
}

func TestLoadToleratesMissingExercisesDir(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters:\n  - chapters/empty\n")
	writeFile(t, filepath.Join(root, "chapters", "empty", "chapter.yaml"), "id: empty\ntitle: t\ntopology: topology.clab.yml\n")
	writeFile(t, filepath.Join(root, "chapters", "empty", "topology.clab.yml"), "name: lab-empty\ntopology:\n  nodes: {}\n")

	lib, err := Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(lib.Chapters) != 1 || len(lib.Chapters[0].Exercises) != 0 {
		t.Fatalf("chapters = %+v", lib.Chapters)
	}
}

func writeFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
