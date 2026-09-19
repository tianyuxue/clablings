package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/watch"
)

func TestLifecycleAdapterSatisfiesWatchEngine(t *testing.T) {
	var _ watch.Engine = newLifecycleAdapter(engine.New("containerlab", "docker"))
}

func TestLifecycleAdapterNoTopologySkipsEngine(t *testing.T) {
	fr := &stubRunner{}
	eng := engine.NewWithRunner(fr, "containerlab", "docker")
	a := newLifecycleAdapter(eng)
	ch := &content.Chapter{ID: "c1"} // no TopologyPath
	if err := a.Deploy(context.Background(), ch); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if err := a.Destroy(context.Background(), ch); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if len(fr.calls) != 0 {
		t.Fatalf("engine calls = %v, want none", fr.calls)
	}
}

func TestLifecycleAdapterNoTopologyRunsSetup(t *testing.T) {
	fr := &stubRunner{}
	eng := engine.NewWithRunner(fr, "containerlab", "docker")
	a := newLifecycleAdapter(eng)
	ch := &content.Chapter{ID: "c1", Setup: []string{"exit 1"}} // no TopologyPath
	err := a.Deploy(context.Background(), ch)
	if err == nil || !strings.Contains(err.Error(), "setup failed") {
		t.Fatalf("err = %v, want setup failed", err)
	}
	if len(fr.calls) != 0 {
		t.Fatalf("engine calls = %v, want none", fr.calls)
	}
}
