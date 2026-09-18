package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

// binOr returns the configured value, falling back to the default command name.
func binOr(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

// lifecycleAdapter adapts engine.Engine to watch.Engine.
type lifecycleAdapter struct{ eng *engine.Engine }

func newLifecycleAdapter(eng *engine.Engine) *lifecycleAdapter {
	return &lifecycleAdapter{eng: eng}
}

func (a *lifecycleAdapter) Deploy(ctx context.Context, ch *content.Chapter) error {
	dctx, cancel := context.WithTimeout(ctx, chapterTimeout(ch))
	defer cancel()
	res, err := a.eng.Deploy(dctx, ch.TopologyPath, chapterTimeout(ch))
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab deploy failed (code=%d): %s", res.Code, res.Stderr)
	}
	if err := runHostCommands(dctx, ch.Setup); err != nil {
		return fmt.Errorf("chapter %s setup failed: %w", ch.ID, err)
	}
	return nil
}

func (a *lifecycleAdapter) Destroy(ctx context.Context, ch *content.Chapter) error {
	if err := runHostCommands(ctx, ch.Teardown); err != nil {
		fmt.Fprintf(os.Stderr, "warning: chapter %s teardown failed: %v\n", ch.ID, err)
	}
	dctx, cancel := context.WithTimeout(ctx, chapterTimeout(ch))
	defer cancel()
	res, err := a.eng.Destroy(dctx, ch.TopologyPath)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab destroy failed (code=%d): %s", res.Code, res.Stderr)
	}
	return nil
}

// runHostCommands runs chapter-level host commands in order, stopping at the first failure.
// It maps context cancellation to ctx.Err() and bounds the wait for orphaned descendants.
func runHostCommands(ctx context.Context, cmds []string) error {
	for _, c := range cmds {
		cmd := exec.CommandContext(ctx, "sh", "-lc", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.WaitDelay = 5 * time.Second
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("%q: %w", c, err)
		}
	}
	return nil
}

func chapterTimeout(ch *content.Chapter) time.Duration {
	if d, err := time.ParseDuration(ch.Deploy.Timeout); err == nil && d > 0 {
		return d
	}
	return 2 * time.Minute
}

// makeVerifier combines assertions into a watch.Verifier. File assertions run
// first (sentinel gate); if any fails, the rest are skipped. P5 inserts the
// code-implement build step here.
func makeVerifier(eng *engine.Engine) func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
	return func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
		var topo *engine.Topology
		if ex.ChapterRef != nil {
			topo, _ = engine.ParseTopology(ex.ChapterRef.TopologyPath)
		}
		fileResults := assert.RunFiltered(ctx, eng, topo, ex, func(a content.Assertion) bool {
			return a.Type == "file"
		})
		for _, r := range fileResults {
			if !r.Passed {
				return fileResults, nil
			}
		}
		rest := assert.RunFiltered(ctx, eng, topo, ex, func(a content.Assertion) bool {
			return a.Type != "file"
		})
		return append(fileResults, rest...), nil
	}
}
