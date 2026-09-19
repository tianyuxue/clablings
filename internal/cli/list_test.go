package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

func sampleLibrary() *content.Library {
	ex1 := &content.Exercise{ID: "01_mtu", Title: "修复 OSPF 邻居卡在 ExStart", Type: content.TypeConfigFix}
	ex1.Chapter = "01-ospf-mtu"
	ex2 := &content.Exercise{ID: "01_arp", Title: "实现 ARP responder", Type: content.TypeCodeImplement}
	ex2.Chapter = "02-arp"
	ch1 := &content.Chapter{ID: "01-ospf-mtu", Title: "OSPF MTU 不匹配", Exercises: []*content.Exercise{ex1}}
	ch2 := &content.Chapter{ID: "02-arp", Title: "ARP responder", Exercises: []*content.Exercise{ex2}}
	return &content.Library{
		Manifest: content.Manifest{Version: 1},
		Chapters: []*content.Chapter{ch1, ch2},
		Order:    []*content.Exercise{ex1, ex2},
		ByID:     map[string]*content.Exercise{"01-ospf-mtu/01_mtu": ex1, "02-arp/01_arp": ex2},
	}
}

func TestFormatLibraryText(t *testing.T) {
	st := &state.State{Version: 1, Exercises: map[string]state.ExerciseState{
		"01-ospf-mtu/01_mtu": {Status: state.StatusPassed},
	}}
	out := formatLibraryText(sampleLibrary(), st)

	if !strings.Contains(out, "01-ospf-mtu OSPF MTU 不匹配") {
		t.Fatalf("missing chapter line:\n%s", out)
	}
	if !strings.Contains(out, "✓ 01_mtu 修复 OSPF 邻居卡在 ExStart") {
		t.Fatalf("missing passed marker:\n%s", out)
	}
	if !strings.Contains(out, "· 01_arp 实现 ARP responder") {
		t.Fatalf("missing todo marker:\n%s", out)
	}
}

func writeTempLibrary(t *testing.T, root string) {
	t.Helper()
	writeCLIFile(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters:\n  - chapters/c1\n")
	writeCLIFile(t, filepath.Join(root, "chapters", "c1", "chapter.yaml"), "id: c1\ntitle: 章一\ntopology: topology.clab.yml\n")
	writeCLIFile(t, filepath.Join(root, "chapters", "c1", "topology.clab.yml"), "name: lab-c1\ntopology:\n  nodes: {}\n")
	writeCLIFile(t, filepath.Join(root, "chapters", "c1", "exercises", "e1", "exercise.yaml"), "id: e1\ntitle: 题一\ntype: config-fix\n")
}

func writeCLIFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListCommandText(t *testing.T) {
	root := t.TempDir()
	writeTempLibrary(t, root)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list", "--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "c1 章一") {
		t.Fatalf("missing chapter line:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "· e1 题一") {
		t.Fatalf("missing exercise line:\n%s", out.String())
	}
}

func TestListCommandJSON(t *testing.T) {
	root := t.TempDir()
	writeTempLibrary(t, root)

	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"list", "--root", root, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	var got struct {
		Root     string `json:"root"`
		Chapters []struct {
			ID        string `json:"id"`
			Exercises []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"exercises"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v\n%s", err, out.String())
	}
	if got.Root != root {
		t.Fatalf("root = %q, want %q", got.Root, root)
	}
	if len(got.Chapters) != 1 || got.Chapters[0].ID != "c1" {
		t.Fatalf("chapters = %+v", got.Chapters)
	}
	if len(got.Chapters[0].Exercises) != 1 {
		t.Fatalf("exercises = %+v", got.Chapters[0].Exercises)
	}
	ex := got.Chapters[0].Exercises[0]
	if ex.ID != "e1" || ex.Status != "todo" {
		t.Fatalf("exercise = %+v", ex)
	}
}

func TestListCommandCorruptStateWarns(t *testing.T) {
	root := t.TempDir()
	writeTempLibrary(t, root)
	writeCLIFile(t, filepath.Join(root, state.StateFileName), "{not json")

	cmd := NewRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"list", "--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "e1 题一") {
		t.Fatalf("stdout missing exercise line:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "warning:") {
		t.Fatalf("stderr missing warning:\n%s", errOut.String())
	}
}

func TestResolveRootPrecedence(t *testing.T) {
	root := t.TempDir()
	writeTempLibrary(t, root)
	t.Setenv("CLABLINGS_ROOT", root)

	got, err := resolveRoot("")
	if err != nil {
		t.Fatalf("resolveRoot(\"\"): %v", err)
	}
	if got != root {
		t.Fatalf("resolveRoot(\"\") = %q, want %q", got, root)
	}

	got, err = resolveRoot(root)
	if err != nil {
		t.Fatalf("resolveRoot(root): %v", err)
	}
	if got != root {
		t.Fatalf("resolveRoot(root) = %q, want %q", got, root)
	}
}

func TestListJSONIncludesMetadata(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: STP\ndifficulty: 3\ntags: [l2, stp]\ntopology: topology.clab.yml\n")
	write("chapters/c1/topology.clab.yml", "name: lab-c1\ntopology:\n  nodes: {}\n")
	write("chapters/c1/exercises/e1/exercise.yaml", "id: e1\ntitle: Root election\ntype: config-fix\ndifficulty: 2\ntags: [stp]\nassertions:\n  - name: x\n    type: exec\n    node: n1\n    cmd: \"true\"\n")

	cmd := newListCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	var view struct {
		Chapters []struct {
			Difficulty int      `json:"difficulty"`
			Tags       []string `json:"tags"`
			Exercises  []struct {
				Difficulty int      `json:"difficulty"`
				Tags       []string `json:"tags"`
			} `json:"exercises"`
		} `json:"chapters"`
	}
	if err := json.Unmarshal(out.Bytes(), &view); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if len(view.Chapters) != 1 || view.Chapters[0].Difficulty != 3 || len(view.Chapters[0].Tags) != 2 {
		t.Fatalf("chapter metadata = %+v", view.Chapters)
	}
	if len(view.Chapters[0].Exercises) != 1 || view.Chapters[0].Exercises[0].Difficulty != 2 || len(view.Chapters[0].Exercises[0].Tags) != 1 {
		t.Fatalf("exercise metadata = %+v", view.Chapters[0].Exercises)
	}
}
