package assert

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

const notDone = "I AM NOT DONE"

func TestFileHostNotContainsSentinel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("int main(){return 0;}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &content.Exercise{ID: "01", Dir: dir}
	a := &content.Assertion{Type: "file", Source: "main.c"}
	res := runFile(context.Background(), &fakeExec{}, ex, a)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}
}

func TestFileHostContainsSentinelFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("// "+notDone+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &content.Exercise{ID: "01", Dir: dir}
	a := &content.Assertion{Type: "file", Source: "main.c"}
	res := runFile(context.Background(), &fakeExec{}, ex, a)
	if res.Passed {
		t.Fatalf("want failure: %+v", res)
	}
}

func TestFileNodeContains(t *testing.T) {
	fe := &fakeExec{out: map[string]engine.Result{
		"cat /etc/frr/frr.conf": {Stdout: "router ospf\n"},
	}}
	ex := &content.Exercise{ID: "01", ChapterRef: &content.Chapter{TopologyName: "lab"}}
	a := &content.Assertion{Type: "file", Node: "r1", Path: "/etc/frr/frr.conf", Contains: "router ospf"}
	res := runFile(context.Background(), fe, ex, a)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}
}
