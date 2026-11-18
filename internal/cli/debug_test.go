package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/state"
)

func writeDebugLibrary(t *testing.T) (root string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(rel, "exercise.yaml") && !strings.Contains(s, "\nbrief:") {
			s = strings.Replace(s, "\ntype:", "\nbrief: 测试题目\ntype:", 1)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: t\ntopology: topology.clab.yml\n")
	write("chapters/c1/topology.clab.yml", "name: static\ntopology:\n  nodes:\n    r1:\n      kind: linux\n      image: alpine:3.20\n")
	write("chapters/c1/exercises/e1/exercise.yaml", "id: e1\ntitle: t\ntype: config-fix\nassertions:\n  - name: up\n    type: exec\n    node: r1\n    cmd: \"true\"\n")
	return root
}

func TestDebugLaunchesClabTuiWithDerivedTopology(t *testing.T) {
	root := writeDebugLibrary(t)
	binDir := t.TempDir()
	rec := filepath.Join(binDir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + rec + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "clab-tui"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Chdir(filepath.Join(root, "chapters", "c1"))

	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("debug: %v", err)
	}

	got, err := os.ReadFile(rec)
	if err != nil {
		t.Fatalf("clab-tui was not invoked: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 || lines[0] != "--file" {
		t.Fatalf("clab-tui args = %q", lines)
	}
	derived := lines[1]
	if filepath.Dir(derived) != filepath.Join(root, "chapters", "c1") {
		t.Fatalf("derived dir = %q", filepath.Dir(derived))
	}
	topo, err := engine.ParseTopology(derived)
	if err != nil {
		t.Fatalf("derived topology: %v", err)
	}
	if topo.Name == "static" || !strings.HasSuffix(topo.Name, "-static") {
		t.Fatalf("derived name = %q, want instance-prefixed", topo.Name)
	}
}

func TestDebugFromExerciseSubdirUsesChapter(t *testing.T) {
	root := writeDebugLibrary(t)
	binDir := t.TempDir()
	rec := filepath.Join(binDir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + rec + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "clab-tui"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Chdir(filepath.Join(root, "chapters", "c1", "exercises", "e1"))

	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("debug from exercise subdir: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(readFile(t, rec)), "\n")
	if len(lines) != 2 || lines[0] != "--file" {
		t.Fatalf("clab-tui args = %q", lines)
	}
	if filepath.Dir(lines[1]) != filepath.Join(root, "chapters", "c1") {
		t.Fatalf("derived dir = %q, want chapter dir", filepath.Dir(lines[1]))
	}
}

func TestDebugUsesCurrentExerciseFromState(t *testing.T) {
	root := writeDebugLibrary(t)
	st, err := state.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	st.SetCurrent("c1/e1")
	if err := st.Save(root); err != nil {
		t.Fatal(err)
	}

	binDir := t.TempDir()
	rec := filepath.Join(binDir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + rec + "\n"
	if err := os.WriteFile(filepath.Join(binDir, "clab-tui"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Chdir(root)

	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("debug from library root with state: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(readFile(t, rec)), "\n")
	if len(lines) != 2 || lines[0] != "--file" {
		t.Fatalf("clab-tui args = %q", lines)
	}
	if filepath.Dir(lines[1]) != filepath.Join(root, "chapters", "c1") {
		t.Fatalf("derived dir = %q, want current exercise's chapter", filepath.Dir(lines[1]))
	}
}

func TestDebugErrorsWhenNotInChapter(t *testing.T) {
	root := writeDebugLibrary(t)
	t.Chdir(root)
	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "chapter") {
		t.Fatalf("err = %v, want chapter-dir error", err)
	}
}
func TestDebugErrorsOnChapterWithoutTopology(t *testing.T) {
	root := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(rel, "exercise.yaml") && !strings.Contains(s, "\nbrief:") {
			s = strings.Replace(s, "\ntype:", "\nbrief: 测试题目\ntype:", 1)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: t\n")
	t.Setenv("PATH", t.TempDir())
	t.Chdir(filepath.Join(root, "chapters", "c1"))
	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no topology") {
		t.Fatalf("err = %v, want no-topology error", err)
	}
}

func TestDebugErrorsWhenClabTuiMissing(t *testing.T) {
	root := writeDebugLibrary(t)
	t.Setenv("PATH", t.TempDir())
	t.Chdir(filepath.Join(root, "chapters", "c1"))
	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "clab-tui") {
		t.Fatalf("err = %v, want clab-tui-not-found error", err)
	}
}

func TestDebugSudoLaunchesViaSudo(t *testing.T) {
	root := writeDebugLibrary(t)
	binDir := t.TempDir()
	rec := filepath.Join(binDir, "args.txt")
	sudoMarker := filepath.Join(binDir, "sudo-called")
	if err := os.WriteFile(filepath.Join(binDir, "clab-tui"),
		[]byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > "+rec+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "sudo"),
		[]byte("#!/bin/sh\necho called > "+sudoMarker+"\nexec \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Chdir(filepath.Join(root, "chapters", "c1"))

	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root, "--sudo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("debug --sudo: %v", err)
	}
	if _, err := os.Stat(sudoMarker); err != nil {
		t.Fatalf("sudo was not invoked: %v", err)
	}
	got, err := os.ReadFile(rec)
	if err != nil {
		t.Fatalf("clab-tui was not invoked through sudo: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(got)), "\n")
	if len(lines) != 2 || lines[0] != "--file" {
		t.Fatalf("clab-tui args = %q", lines)
	}
}

func TestDebugSudoErrorsWhenSudoMissing(t *testing.T) {
	root := writeDebugLibrary(t)
	binDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(binDir, "clab-tui"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir)
	t.Chdir(filepath.Join(root, "chapters", "c1"))
	cmd := newDebugCmd()
	cmd.SetArgs([]string{"--root", root, "--sudo"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "sudo") {
		t.Fatalf("err = %v, want sudo-not-found error", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
