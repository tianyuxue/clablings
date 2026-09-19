package cli

import (
	"context"
	"os"
	"path/filepath"
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

type psRunner struct {
	calls [][]string
	out   string
}

func (r *psRunner) Run(_ context.Context, name string, args ...string) (engine.Result, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if name == "docker" && len(args) > 0 && args[0] == "ps" {
		return engine.Result{Code: 0, Stdout: r.out}, nil
	}
	return engine.Result{}, nil
}

func TestLifecycleAdapterReusesRunningLab(t *testing.T) {
	dir := t.TempDir()
	topo := filepath.Join(dir, "topology.clab.yml")
	if err := os.WriteFile(topo, []byte("name: x\ntopology:\n  nodes: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr := &psRunner{out: "clab-x-r1\n"}
	eng := engine.NewWithRunner(fr, "containerlab", "docker")
	a := newLifecycleAdapter(eng)
	ch := &content.Chapter{ID: "c1", TopologyPath: topo, LabName: "x"}
	if err := a.Deploy(context.Background(), ch); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	for _, c := range fr.calls {
		if len(c) > 1 && c[0] == "containerlab" && c[1] == "deploy" {
			t.Fatalf("must not redeploy a running lab: %v", fr.calls)
		}
	}
}
