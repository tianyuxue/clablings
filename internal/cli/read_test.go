package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReadLibrary(t *testing.T) string {
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
	write("chapters/c1/chapter.yaml", "id: c1\ntitle:\n  en: Switching\n  zh: 交换\nobjectives:\n  en: [obj a]\n  zh: [目标甲]\ntopology: topology.clab.yml\n")
	write("chapters/c1/topology.clab.yml", "name: lab-c1\ntopology:\n  nodes: {}\n")
	write("chapters/c1/concept.en.md", "# Switching\n\nWhy we need L2 forwarding.\n")
	write("chapters/c1/concept.zh.md", "# 交换\n\n为什么需要二层转发。\n")
	write("chapters/c1/reading.en.yaml", "- title: IEEE 802.1D\n  section: \"1\"\n  kind: required\n  intro:\n    en: Overview\n    zh: 概述\n  question:\n    en: Why loops break Ethernet?\n    zh: 环路为何瘫痪以太网？\n")
	return root
}

func TestReadShowsLocalizedConceptAndReading(t *testing.T) {
	root := writeReadLibrary(t)
	t.Setenv("CLABLINGS_LANG", "zh")
	cmd := newReadCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root, "c1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read: %v", err)
	}
	got := out.String()
	for _, want := range []string{"交换", "为什么需要二层转发。", "目标甲", "IEEE 802.1D", "概述", "环路为何瘫痪以太网？"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestReadUnknownChapter(t *testing.T) {
	root := writeReadLibrary(t)
	cmd := newReadCmd()
	cmd.SetArgs([]string{"--root", root, "nope"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown chapter") {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFallsBackToEnglishConcept(t *testing.T) {
	root := writeReadLibrary(t)
	t.Setenv("CLABLINGS_LANG", "fr")
	cmd := newReadCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root, "c1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(out.String(), "Why we need L2 forwarding.") {
		t.Fatalf("expected english fallback:\n%s", out.String())
	}
}
