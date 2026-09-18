package assert

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

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
	if err := os.WriteFile(filepath.Join(dir, "main.c"), []byte("// "+Sentinel+"\n"), 0o644); err != nil {
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
		"cat '/etc/frr/frr.conf'": {Stdout: "router ospf\n"},
	}}
	ex := &content.Exercise{ID: "01", ChapterRef: &content.Chapter{TopologyName: "lab"}}
	a := &content.Assertion{Type: "file", Node: "r1", Path: "/etc/frr/frr.conf", Contains: "router ospf"}
	res := runFile(context.Background(), fe, ex, a)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}
}

func TestRunFileBranches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "has.txt"), []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	ex := &content.Exercise{ID: "01", Dir: dir, ChapterRef: &content.Chapter{TopologyName: "lab"}}

	cases := []struct {
		name       string
		a          content.Assertion
		execRes    engine.Result
		execErr    error
		wantPassed bool
		wantErr    bool
	}{
		{"host contains pass", content.Assertion{Type: "file", Source: "has.txt", Contains: "hello"}, engine.Result{}, nil, true, false},
		{"host contains fail", content.Assertion{Type: "file", Source: "has.txt", Contains: "nope"}, engine.Result{}, nil, false, false},
		{"host not_contains pass", content.Assertion{Type: "file", Source: "has.txt", NotContains: "nope"}, engine.Result{}, nil, true, false},
		{"host not_contains fail", content.Assertion{Type: "file", Source: "has.txt", NotContains: "hello"}, engine.Result{}, nil, false, false},
		{"host sentinel removed", content.Assertion{Type: "file", Source: "has.txt"}, engine.Result{}, nil, true, false},
		{"host read error", content.Assertion{Type: "file", Source: "missing.txt"}, engine.Result{}, nil, false, true},
		{"node contains pass", content.Assertion{Type: "file", Node: "r1", Path: "/f", Contains: "x"}, engine.Result{Stdout: "x"}, nil, true, false},
		{"node sentinel removed", content.Assertion{Type: "file", Node: "r1", Path: "/f"}, engine.Result{Stdout: "clean"}, nil, true, false},
		{"node non-zero code", content.Assertion{Type: "file", Node: "r1", Path: "/f", Contains: "x"}, engine.Result{Code: 1}, nil, false, false},
		{"node exec error", content.Assertion{Type: "file", Node: "r1", Path: "/f"}, engine.Result{}, errors.New("boom"), false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fe := &fakeExec{execErr: c.execErr, out: map[string]engine.Result{"cat '/f'": c.execRes}}
			got := runFile(context.Background(), fe, ex, &c.a)
			if got.Passed != c.wantPassed {
				t.Fatalf("passed = %v, want %v (%+v)", got.Passed, c.wantPassed, got)
			}
			if (got.Err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", got.Err, c.wantErr)
			}
		})
	}
}
