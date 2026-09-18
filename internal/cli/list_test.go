package cli

import (
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
	}
}

func TestFormatLibraryText(t *testing.T) {
	st := &state.State{Version: 1, Exercises: map[string]state.ExerciseState{
		"01-ospf-mtu/01_mtu": {Status: state.StatusPassed},
	}}
	out := formatLibraryText(sampleLibrary(), st)

	if !strings.Contains(out, "01-ospf-mtu OSPF MTU 不匹配") {
		t.Fatalf("缺少章节行:\n%s", out)
	}
	if !strings.Contains(out, "✓ 01_mtu 修复 OSPF 邻居卡在 ExStart") {
		t.Fatalf("缺少通过标记:\n%s", out)
	}
	if !strings.Contains(out, "· 01_arp 实现 ARP responder") {
		t.Fatalf("缺少未开始标记:\n%s", out)
	}
}
