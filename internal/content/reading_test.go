package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadingLocalizedAndFallback(t *testing.T) {
	dir := t.TempDir()
	en := "- rfc: 2328\n  title: OSPF Version 2\n  section: \"7.3\"\n  section_title: Database Description\n  kind: required\n  url: https://www.rfc-editor.org/rfc/rfc2328\n  intro:\n    en: How databases sync\n    zh: 数据库如何同步\n  question:\n    en: Why is MTU in the DBD?\n    zh: 为什么 DBD 里带 MTU？\n"
	if err := os.WriteFile(filepath.Join(dir, "reading.en.yaml"), []byte(en), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadReading(dir, "zh")
	if err != nil {
		t.Fatalf("LoadReading zh: %v", err)
	}
	if len(got) != 1 || got[0].RFC != 2328 || got[0].Section != "7.3" {
		t.Fatalf("reading = %+v", got)
	}
	if got[0].Intro != "数据库如何同步" || got[0].Question != "为什么 DBD 里带 MTU？" {
		t.Fatalf("localized text = %+v", got[0])
	}

	gotEN, err := LoadReading(dir, "en")
	if err != nil {
		t.Fatalf("LoadReading en: %v", err)
	}
	if gotEN[0].Intro != "How databases sync" {
		t.Fatalf("en intro = %q", gotEN[0].Intro)
	}

	missing, err := LoadReading(t.TempDir(), "en")
	if err != nil || missing != nil {
		t.Fatalf("missing reading: %v %v", missing, err)
	}
}

func TestLoadReadingPrefersLocaleFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reading.en.yaml"), []byte("- rfc: 1\n  title: EN\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reading.zh.yaml"), []byte("- rfc: 1\n  title: ZH\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadReading(dir, "zh")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Title != "ZH" {
		t.Fatalf("reading = %+v", got)
	}
}
