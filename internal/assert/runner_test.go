package assert

import (
	"context"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

type fakeExec struct {
	execs []string
	out   map[string]engine.Result
	iface map[string][]engine.InspectInterface
}

func (f *fakeExec) Exec(_ context.Context, container, cmd string) (engine.Result, error) {
	f.execs = append(f.execs, container+": "+cmd)
	if r, ok := f.out[cmd]; ok {
		return r, nil
	}
	return engine.Result{}, nil
}

func (f *fakeExec) InspectInterfaces(context.Context, string) (map[string][]engine.InspectInterface, error) {
	return f.iface, nil
}

func TestContainerNameFromExercise(t *testing.T) {
	ex := &content.Exercise{ChapterRef: &content.Chapter{TopologyName: "clablings-ospf-mtu"}}
	if got := containerName(ex, "r1"); got != "clab-clablings-ospf-mtu-r1" {
		t.Fatalf("containerName = %q", got)
	}
}
