package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xansi "github.com/charmbracelet/x/ansi"
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
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: 交换\nobjectives: [目标甲]\ntopology: topology.clab.yml\n")
	write("chapters/c1/topology.clab.yml", "name: lab-c1\ntopology:\n  nodes: {}\n")
	write("chapters/c1/concept.md", "# 交换\n\n为什么需要二层转发。\n")
	write("chapters/c1/reading.yaml", "- title: IEEE 802.1D\n  section: \"1\"\n  kind: required\n  intro: 概述\n  question: 环路为何瘫痪以太网？\n")
	return root
}

func TestReadShowsChineseConceptAndReading(t *testing.T) {
	root := writeReadLibrary(t)
	cmd := newReadCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root, "c1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read: %v", err)
	}
	got := out.String()
	for _, want := range []string{"交换", "为什么需要二层转发。", "目标甲", "## 目标", "## 阅读", "导读", "问题", "必读", "IEEE 802.1D", "概述", "环路为何瘫痪以太网？"} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

func TestReadDoesNotFallbackToLegacyConcept(t *testing.T) {
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
	write("chapters/c1/chapter.yaml", "id: c1\ntitle: 交换\nconcept: ignored\n")
	write("chapters/c1/concept.en.md", "# English\n\nEnglish fallback\n")
	cmd := newReadCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--root", root, "c1"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(out.String(), "English fallback") {
		t.Fatalf("unexpected legacy fallback: %s", out.String())
	}
}

func TestSpaceHanChars(t *testing.T) {
	in := "# 标题\n\n正常中文，`code 中文` 与 **加粗**。\n\n```sh\necho 中文\n```\n\n[中文](https://example.com/中文)\n"
	want := "# 标 题\n\n正 常 中 文，`code 中文` 与 **加 粗**。\n\n```sh\necho 中文\n```\n\n[中 文](https://example.com/中文)\n"
	if got := spaceHanChars(in); got != want {
		t.Fatalf("spaceHanChars:\n got %q\nwant %q", got, want)
	}
}

func TestRemoveHanSpaces(t *testing.T) {
	for _, tt := range []struct {
		in, want string
	}{
		{in: "中 文 and 中 文", want: "中文 and 中文"},
		{in: "中 \x1b[31m文\x1b[0m", want: "中\x1b[31m文\x1b[0m"},
	} {
		if got := removeHanSpaces(tt.in); got != tt.want {
			t.Fatalf("removeHanSpaces(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWrapRendered(t *testing.T) {
	long := "交换机底层基于数据帧的字节序列进行转发决策，它必须明确当前帧属于哪个广播域，例如 802.1Q 标签。"
	got := removeHanSpaces(wrapRendered(spaceHanChars(long), 20))
	for _, line := range strings.Split(got, "\n") {
		if width := xansi.StringWidthWc(xansi.Strip(line)); width > 20 {
			t.Fatalf("line width %d > 20: %q\n%s", width, line, got)
		}
	}
	if !strings.Contains(strings.ReplaceAll(got, "\n", ""), "802.1Q") {
		t.Fatalf("token split: %q", got)
	}

	// ANSI styling must survive wrapping.
	styled := "\x1b[31m" + spaceHanChars(long) + "\x1b[0m"
	wrapped := removeHanSpaces(wrapRendered(styled, 20))
	if !strings.Contains(wrapped, "\x1b[31m") || !strings.Contains(wrapped, "\x1b[0m") {
		t.Fatalf("ANSI codes lost: %q", wrapped)
	}

	// Code lines and URLs are left to the terminal rather than broken.
	code := "    ip link set eth1 master br0 && ip link set eth2 master br0"
	if got := wrapRendered(code, 10); got != code {
		t.Fatalf("code line wrapped: %q", got)
	}
	url := "  - https://www.rfc-editor.org/rfc/rfc7432#section-8.2"
	if got := wrapRendered(url, 10); got != url {
		t.Fatalf("URL wrapped: %q", got)
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
