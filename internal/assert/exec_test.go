package assert

import (
	"context"
	"errors"
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

func TestRunExecBranches(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}
	zero := 0
	cases := []struct {
		name       string
		match      *content.Match
		res        engine.Result
		execErr    error
		wantPassed bool
		wantErr    bool
	}{
		{"runner error", nil, engine.Result{}, errors.New("boom"), false, true},
		{"no match zero exit", nil, engine.Result{Code: 0}, nil, true, false},
		{"no match nonzero exit", nil, engine.Result{Code: 1}, nil, false, false},
		{"exit code ok", &content.Match{ExitCode: &zero}, engine.Result{Code: 0}, nil, true, false},
		{"exit code mismatch", &content.Match{ExitCode: &zero}, engine.Result{Code: 1}, nil, false, false},
		{"json pass", &content.Match{JSON: "$.state", TextMatch: content.TextMatch{Equals: "Full"}}, engine.Result{Stdout: `{"state":"Full"}`}, nil, true, false},
		{"json fail", &content.Match{JSON: "$.state", TextMatch: content.TextMatch{Equals: "Full"}}, engine.Result{Stdout: `{"state":"Down"}`}, nil, false, false},
		{"stdout pass", &content.Match{Stdout: &content.TextMatch{Contains: "ok"}}, engine.Result{Stdout: "ok\n"}, nil, true, false},
		{"stdout fail", &content.Match{Stdout: &content.TextMatch{Contains: "ok"}}, engine.Result{Stdout: "no\n"}, nil, false, false},
		{"inline match pass", &content.Match{TextMatch: content.TextMatch{Contains: "ok"}}, engine.Result{Stdout: "ok\n"}, nil, true, false},
		{"regex error", &content.Match{TextMatch: content.TextMatch{Regex: "("}}, engine.Result{Stdout: "ok"}, nil, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fe := &fakeExec{execErr: c.execErr, out: map[string]engine.Result{"cmd": c.res}}
			a := &content.Assertion{Type: "exec", Node: "n", Cmd: "cmd", Match: c.match}
			got := runExec(context.Background(), fe, ex, a)
			if got.Passed != c.wantPassed {
				t.Fatalf("passed = %v, want %v (%+v)", got.Passed, c.wantPassed, got)
			}
			if (got.Err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr %v", got.Err, c.wantErr)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	if got := firstLine("a\nb\n"); got != `"a" ...` {
		t.Fatalf("firstLine multiline = %q", got)
	}
	if got := firstLine("solo"); got != `"solo"` {
		t.Fatalf("firstLine single = %q", got)
	}
}
