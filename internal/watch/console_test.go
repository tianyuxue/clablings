package watch

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
)

func TestCRLFWriterTransformsNewlines(t *testing.T) {
	var buf bytes.Buffer
	w := NewCRLFWriter(&buf)
	if _, err := w.Write([]byte("a\nb\n")); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "a\r\nb\r\n" {
		t.Fatalf("got %q", got)
	}
}

func TestCRLFWriterDoesNotDoubleCR(t *testing.T) {
	var buf bytes.Buffer
	w := NewCRLFWriter(&buf)
	_, _ = w.Write([]byte("a\r\n"))
	_, _ = w.Write([]byte("b\n"))
	if got := buf.String(); got != "a\r\nb\r\n" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteHints(t *testing.T) {
	var buf bytes.Buffer
	WriteHints(&buf, &content.Exercise{Hints: []string{"one", "two"}})
	out := buf.String()
	if !strings.Contains(out, "1. one") || !strings.Contains(out, "2. two") {
		t.Fatalf("out = %q", out)
	}

	buf.Reset()
	WriteHints(&buf, &content.Exercise{})
	if !strings.Contains(buf.String(), "no hints") {
		t.Fatalf("out = %q", buf.String())
	}
}

func TestWriteLegend(t *testing.T) {
	var buf bytes.Buffer
	WriteLegend(&buf)
	if !strings.Contains(buf.String(), "[h] hint") || !strings.Contains(buf.String(), "[c] concept") || !strings.Contains(buf.String(), "[x] clean quit") {
		t.Fatalf("legend = %q", buf.String())
	}
}
