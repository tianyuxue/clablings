//go:build integration

package reset

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func TestIntegrationDeployAndRedeploy(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	dir := t.TempDir()
	lab := fmt.Sprintf("clablings-it-reset-%d", os.Getpid())
	topo := filepath.Join(dir, "it.clab.yml")
	yaml := fmt.Sprintf(`
name: %s
topology:
  nodes:
    n1:
      kind: linux
      image: alpine:3.20
    n2:
      kind: linux
      image: alpine:3.20
  links:
    - endpoints: ["n1:eth1", "n2:eth1"]
`, lab)
	if err := os.WriteFile(topo, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New("containerlab", "docker")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	deployOK := false
	t.Cleanup(func() {
		cctx, c := context.WithTimeout(context.Background(), time.Minute)
		defer c()
		res, err := eng.Destroy(cctx, topo, "")
		if err != nil {
			if deployOK {
				t.Errorf("cleanup destroy: %v", err)
			}
			return
		}
		if deployOK && res.Code != 0 {
			t.Errorf("cleanup destroy code=%d: %s", res.Code, res.Stderr)
		}
	})

	res, err := eng.Deploy(ctx, topo, "", 2*time.Minute)
	if err != nil {
		t.Fatalf("deploy: %v", err)
	}
	if res.Code != 0 {
		t.Fatalf("deploy code=%d: %s", res.Code, res.Stderr)
	}
	deployOK = true

	nodes, err := eng.Inspect(ctx, topo, "")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(nodes[lab]) != 2 {
		t.Fatalf("inspect nodes = %v", nodes[lab])
	}

	// Exercise the reset hard path (adapter: teardown -> redeploy -> setup).
	ch := &content.Chapter{ID: "it", TopologyPath: topo}
	ex := &content.Exercise{ID: "e", Chapter: "it", Type: content.TypeConfigFix, ChapterRef: ch}
	r := New(Options{Root: dir, Engine: FromEngine(eng)})
	if err := r.Hard(ctx, ex); err != nil {
		t.Fatalf("hard reset: %v", err)
	}

	// Break the topology inside the container, then verify hard reset recovers it.
	container := engine.ContainerName(lab, "n1")
	if out, err := exec.CommandContext(ctx, "docker", "exec", container, "ip", "link", "del", "eth1").CombinedOutput(); err != nil {
		t.Fatalf("del link: %v %s", err, out)
	}
	if err := r.Hard(ctx, ex); err != nil {
		t.Fatalf("hard reset after break: %v", err)
	}
	if out, err := exec.CommandContext(ctx, "docker", "exec", container, "ip", "link", "show", "eth1").CombinedOutput(); err != nil {
		t.Fatalf("eth1 not restored: %v %s", err, out)
	}
}
