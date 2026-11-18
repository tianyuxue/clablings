package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReading(t *testing.T) {
	dir := t.TempDir()
	body := "- rfc: 2328\n  title: OSPF Version 2\n  section: \"7.3\"\n  kind: required\n  url: https://www.rfc-editor.org/rfc/rfc2328\n  intro: 数据库如何同步\n  question: 为什么 DBD 里带 MTU？\n"
	if err := os.WriteFile(filepath.Join(dir, "reading.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReading(dir)
	if err != nil {
		t.Fatalf("LoadReading: %v", err)
	}
	if len(got) != 1 || got[0].RFC != 2328 || got[0].Section != "7.3" {
		t.Fatalf("reading = %+v", got)
	}
	if got[0].Intro != "数据库如何同步" || got[0].Question != "为什么 DBD 里带 MTU？" {
		t.Fatalf("text = %+v", got[0])
	}
}

func TestLoadReadingMissing(t *testing.T) {
	got, err := LoadReading(t.TempDir())
	if err != nil || got != nil {
		t.Fatalf("missing reading: %v %v", got, err)
	}
}

func TestLoadReadingIgnoresLegacyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reading.en.yaml"), []byte("- rfc: 1\n  title: EN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReading(dir)
	if err != nil || got != nil {
		t.Fatalf("legacy file must be ignored: %v %v", got, err)
	}
}
