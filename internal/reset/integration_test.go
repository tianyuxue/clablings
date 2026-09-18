//go:build integration

package reset

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/engine"
)

func TestIntegrationDeployAndRedeploy(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	dir := t.TempDir()
	topo := filepath.Join(dir, "it.clab.yml")
	if err := os.WriteFile(topo, []byte(`
name: clablings-it-reset
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
`), 0o644); err != nil {
		t.Fatal(err)
	}
	eng := engine.New("containerlab", "docker")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if _, err := eng.Deploy(ctx, topo, 2*time.Minute); err != nil {
		t.Fatalf("deploy: %v", err)
	}
	t.Cleanup(func() {
		cctx, c := context.WithTimeout(context.Background(), time.Minute)
		defer c()
		_, _ = eng.Destroy(cctx, topo)
	})

	nodes, err := eng.Inspect(ctx, topo)
	if err != nil || len(nodes["clablings-it-reset"]) != 2 {
		t.Fatalf("inspect: nodes=%v err=%v", nodes, err)
	}
	if _, err := eng.Redeploy(ctx, topo, 2*time.Minute); err != nil {
		t.Fatalf("redeploy: %v", err)
	}

	// Break the topology inside the container, then verify hard reset recovers it.
	container := engine.ContainerName("clablings-it-reset", "n1")
	if out, err := exec.Command("docker", "exec", container, "ip", "link", "del", "eth1").CombinedOutput(); err != nil {
		t.Fatalf("del link: %v %s", err, out)
	}
	if _, err := eng.Redeploy(ctx, topo, 2*time.Minute); err != nil {
		t.Fatalf("redeploy after break: %v", err)
	}
	if out, err := exec.Command("docker", "exec", container, "ip", "link", "show", "eth1").CombinedOutput(); err != nil {
		t.Fatalf("eth1 not restored: %v %s", err, out)
	}
}
