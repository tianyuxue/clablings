//go:build integration

package headless

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/reset"
)

func TestIntegrationRunAll(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "clablings.yaml"), "version: 1\nchapters:\n  - chapters/c1\n")
	chDir := filepath.Join(root, "chapters", "c1")
	mustWrite(t, filepath.Join(chDir, "chapter.yaml"), "id: c1\ntitle: t\ntopology: topology.clab.yml\n")
	mustWrite(t, filepath.Join(chDir, "topology.clab.yml"), "name: clablings-it-headless\ntopology:\n  nodes:\n    n1:\n      kind: linux\n      image: alpine:3.20\n")
	exDir := filepath.Join(chDir, "exercises", "e1")
	mustWrite(t, filepath.Join(exDir, "exercise.yaml"), "id: e1\ntitle: t\ntype: config-fix\nassertions:\n  - name: host file exists\n    type: file\n    source: ok.txt\n")
	mustWrite(t, filepath.Join(exDir, "ok.txt"), "ready\n")

	lib, err := content.Load(root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	eng := engine.New("containerlab", "docker")
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng)})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var seen int
	err = RunAll(ctx, lib, &noopLifecycle{eng: eng}, res,
		func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
			return assert.RunExercise(ctx, eng, nil, ex), nil
		},
		func(id string, results []assert.Result) {
			seen++
			if len(results) == 0 || !results[0].Passed {
				t.Fatalf("exercise %s did not pass: %+v", id, results)
			}
		})
	if err != nil {
		t.Fatalf("RunAll: %v", err)
	}
	if seen != 1 {
		t.Fatalf("seen = %d", seen)
	}
}

func mustWrite(t *testing.T, path, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

// noopLifecycle reuses the real engine's Deploy/Destroy.
type noopLifecycle struct{ eng *engine.Engine }

func (n *noopLifecycle) Deploy(ctx context.Context, ch *content.Chapter) error {
	topo, err := engine.PrepareTopology(ch)
	if err != nil {
		return err
	}
	res, err := n.eng.Deploy(ctx, topo, 2*time.Minute)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("deploy %s failed (code=%d): %s", ch.ID, res.Code, res.Stderr)
	}
	return nil
}

func (n *noopLifecycle) Destroy(ctx context.Context, ch *content.Chapter) error {
	topo, err := engine.PrepareTopology(ch)
	if err != nil {
		return err
	}
	res, err := n.eng.Destroy(ctx, topo)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("destroy %s failed (code=%d): %s", ch.ID, res.Code, res.Stderr)
	}
	return nil
}
