package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLintLibrary(t *testing.T, assertionNode string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: {en: Switching, zh: 交换}\ntopology: topology.clab.yml\n")
	write("chapters/c1/topology.clab.yml", "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n")
	write("chapters/c1/exercises/e1/exercise.yaml", "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: exec\n    node: "+assertionNode+"\n    cmd: \"true\"\n")
	return root
}

func TestLintValidLibrary(t *testing.T) {
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", writeLintLibrary(t, "n1")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lint: %v", err)
	}
	if !strings.Contains(out.String(), "ok:") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintUnknownNode(t *testing.T) {
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", writeLintLibrary(t, "nope")})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure")
	}
	if !strings.Contains(out.String(), "unknown node") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintTopologylessContainerAssertion(t *testing.T) {
	root := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: {en: t, zh: t}\n")
	write("chapters/c1/exercises/e1/exercise.yaml", "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: exec\n    node: n1\n    cmd: \"true\"\n")
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure for container assertion without topology")
	}
	if !strings.Contains(out.String(), "requires a topology") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintTopologylessHostSetupAllowed(t *testing.T) {
	root := t.TempDir()
	write := func(rel, s string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("clablings.yaml", "version: 1\nchapters:\n  - chapters/c1\n")
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: {en: t, zh: t}\nsetup:\n  - \"true\"\n")
	write("chapters/c1/exercises/e1/exercise.yaml", "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: file\n    source: main.c\n")
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lint: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "ok:") {
		t.Fatalf("out = %q", out.String())
	}
}

func writeLintTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, s := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestLintUnknownLinkNode(t *testing.T) {
	root := writeLintTree(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: t, zh: t}\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: link\n    a: n1\n    b: nope\n    state: up\n",
	})
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure for unknown link node")
	}
	if !strings.Contains(out.String(), "unknown node") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintInvalidWait(t *testing.T) {
	root := writeLintTree(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: t, zh: t}\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: exec\n    node: n1\n    cmd: \"true\"\n    wait: abc\n",
	})
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure for invalid wait")
	}
	if !strings.Contains(out.String(), "invalid wait") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintUnknownFileNode(t *testing.T) {
	root := writeLintTree(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: t, zh: t}\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nfiles:\n  - source: main.c\n    node: nope\n    dest: /root/main.c\n",
	})
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure for unknown file node")
	}
	if !strings.Contains(out.String(), "unknown node") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintUnknownAssertionType(t *testing.T) {
	root := writeLintTree(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: t, zh: t}\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: bogus\n    node: n1\n",
	})
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected lint failure for unknown assertion type")
	}
	if !strings.Contains(out.String(), "unknown type") {
		t.Fatalf("out = %q", out.String())
	}
}

func TestLintWaitZeroAllowed(t *testing.T) {
	root := writeLintTree(t, map[string]string{
		"clablings.yaml":                         "version: 1\nchapters:\n  - chapters/c1\n",
		"chapters/c1/chapter.yaml":               "id: c1\ntitle: {en: t, zh: t}\ntopology: topology.clab.yml\n",
		"chapters/c1/topology.clab.yml":          "name: lab-c1\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n",
		"chapters/c1/exercises/e1/exercise.yaml": "id: e1\ntitle: {en: t, zh: t}\ntype: config-fix\nassertions:\n  - name: {en: a, zh: a}\n    type: exec\n    node: n1\n    cmd: \"true\"\n    wait: \"0s\"\n",
	})
	cmd := newLintCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("lint: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "ok:") {
		t.Fatalf("out = %q", out.String())
	}
}
