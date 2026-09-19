package assert

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

type fakeExec struct {
	execs      []string
	out        map[string]engine.Result
	iface      map[string][]engine.InspectInterface
	execErr    error
	ifaceErr   error
	ifaceCalls int
}

func (f *fakeExec) Exec(_ context.Context, container, cmd string) (engine.Result, error) {
	f.execs = append(f.execs, container+": "+cmd)
	if f.execErr != nil {
		return engine.Result{}, f.execErr
	}
	if r, ok := f.out[cmd]; ok {
		return r, nil
	}
	return engine.Result{}, nil
}

func (f *fakeExec) InspectInterfaces(context.Context, string) (map[string][]engine.InspectInterface, error) {
	f.ifaceCalls++
	if f.ifaceErr != nil {
		return nil, f.ifaceErr
	}
	return f.iface, nil
}

func TestContainerNameFromExercise(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "clablings-ospf-mtu"}}
	if got := containerName(ex, "r1"); got != "clab-clablings-ospf-mtu-r1" {
		t.Fatalf("containerName = %q", got)
	}
}

func TestContainerNameFallback(t *testing.T) {
	if got := containerName(&content.Exercise{}, "r1"); got != "r1" {
		t.Fatalf("containerName = %q, want r1", got)
	}
	if got := containerName(&content.Exercise{ChapterRef: &content.Chapter{}}, "r1"); got != "r1" {
		t.Fatalf("containerName = %q, want r1", got)
	}
}

func TestLabName(t *testing.T) {
	if got := labName(&content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}); got != "lab" {
		t.Fatalf("labName = %q", got)
	}
	if got := labName(&content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab", LabName: "inst-lab"}}); got != "inst-lab" {
		t.Fatalf("labName should prefer LabName, got %q", got)
	}
	if got := labName(&content.Exercise{}); got != "" {
		t.Fatalf("labName = %q, want empty", got)
	}
}
