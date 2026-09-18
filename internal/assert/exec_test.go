package assert

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestExecExitCodeAndStdout(t *testing.T) {
	fe := &fakeExec{out: map[string]engine.Result{
		"arping -c 2 -I eth1 10.0.1.1": {Stdout: "reply from 10.0.1.1\n", Code: 0},
	}}
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	code := 0
	a := &content.Assertion{
		Type: "exec", Node: "host1", Cmd: "arping -c 2 -I eth1 10.0.1.1",
		Match: &content.Match{ExitCode: &code, Stdout: &content.TextMatch{Contains: "reply from"}},
	}
	res := runExec(context.Background(), fe, ex, a)
	if !res.Passed {
		t.Fatalf("want pass: %+v", res)
	}
}

func TestExecExitCodeMismatch(t *testing.T) {
	fe := &fakeExec{out: map[string]engine.Result{
		"false": {Code: 1},
	}}
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	code := 0
	a := &content.Assertion{Type: "exec", Node: "n", Cmd: "false", Match: &content.Match{ExitCode: &code}}
	res := runExec(context.Background(), fe, ex, a)
	if res.Passed {
		t.Fatalf("want failure: %+v", res)
	}
	if res.Expected == "" || res.Actual == "" {
		t.Fatalf("Expected/Actual must not be empty: %+v", res)
	}
}
