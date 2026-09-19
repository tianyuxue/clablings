package reset

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestRealEngineSatisfiesResetEngine(t *testing.T) {
	var _ Engine = FromEngine(engine.New("containerlab", "docker"))
}

// fakeRunner records engine calls and returns preset results.
type fakeRunner struct {
	calls   [][]string
	results []engine.Result
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (engine.Result, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	i := len(f.calls) - 1
	if i < len(f.results) {
		return f.results[i], nil
	}
	return engine.Result{}, nil
}

func TestAdapterRedeployCommand(t *testing.T) {
	fr := &fakeRunner{}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	ch := &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml", Deploy: content.Deploy{Timeout: "1m"}}
	if err := a.Redeploy(context.Background(), ch); err != nil {
		t.Fatalf("Redeploy: %v", err)
	}
	want := []string{"containerlab", "redeploy", "-t", "/t.clab.yml", "--cleanup", "--timeout", "1m0s"}
	if len(fr.calls) != 1 || !reflect.DeepEqual(fr.calls[0], want) {
		t.Fatalf("calls = %v, want %v", fr.calls, want)
	}
}

func TestAdapterRedeployIncludesLabName(t *testing.T) {
	fr := &fakeRunner{}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	ch := &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml", LabName: "lab-x", Deploy: content.Deploy{Timeout: "1m"}}
	if err := a.Redeploy(context.Background(), ch); err != nil {
		t.Fatalf("Redeploy: %v", err)
	}
	want := []string{"containerlab", "redeploy", "-t", "/t.clab.yml", "--name", "lab-x", "--cleanup", "--timeout", "1m0s"}
	if !reflect.DeepEqual(fr.calls[0], want) {
		t.Fatalf("calls[0] = %v, want %v", fr.calls[0], want)
	}
}

func TestAdapterRedeployNonZeroCode(t *testing.T) {
	fr := &fakeRunner{results: []engine.Result{{Code: 1, Stderr: "redeploy boom"}}}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	err := a.Redeploy(context.Background(), &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml"})
	if err == nil || !strings.Contains(err.Error(), "redeploy boom") {
		t.Fatalf("err = %v, want contains redeploy boom", err)
	}
}

func TestAdapterTeardownFailureWarnsButContinues(t *testing.T) {
	fr := &fakeRunner{}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	ch := &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml", Teardown: []string{"exit 1"}}
	if err := a.Redeploy(context.Background(), ch); err != nil {
		t.Fatalf("teardown failure must not abort: %v", err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("redeploy not called: %v", fr.calls)
	}
}

func TestAdapterSetupFailure(t *testing.T) {
	fr := &fakeRunner{}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	ch := &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml", Setup: []string{"exit 1"}}
	err := a.Redeploy(context.Background(), ch)
	if err == nil || !strings.Contains(err.Error(), "setup failed") {
		t.Fatalf("err = %v, want setup failed", err)
	}
}

func TestAdapterHostCommandTimeout(t *testing.T) {
	fr := &fakeRunner{}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	ch := &content.Chapter{ID: "c1", TopologyPath: "/t.clab.yml", Setup: []string{"exec sleep 5"}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err := a.Redeploy(ctx, ch)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want DeadlineExceeded", err)
	}
}

func TestAdapterExecNonZeroCode(t *testing.T) {
	fr := &fakeRunner{results: []engine.Result{{Code: 2, Stderr: "exec boom"}}}
	a := FromEngine(engine.NewWithRunner(fr, "containerlab", "docker"))
	err := a.Exec(context.Background(), "c1", "true")
	if err == nil || !strings.Contains(err.Error(), "exec boom") {
		t.Fatalf("err = %v, want contains exec boom", err)
	}
}
