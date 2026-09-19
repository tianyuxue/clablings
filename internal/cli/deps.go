package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/reset"
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
	dctx, cancel := context.WithTimeout(ctx, engine.ChapterDeployTimeout(ch))
	defer cancel()
	topo, err := engine.PrepareTopology(ch)
	if err != nil {
		return err
	}
	res, err := a.eng.Deploy(dctx, topo, engine.ChapterDeployTimeout(ch))
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab deploy failed (code=%d): %s", res.Code, res.Stderr)
	}
	if err := engine.RunHostCommands(dctx, ch.Setup); err != nil {
		return fmt.Errorf("chapter %s setup failed: %w", ch.ID, err)
	}
	return nil
}

func (a *lifecycleAdapter) Destroy(ctx context.Context, ch *content.Chapter) error {
	if err := engine.RunHostCommands(ctx, ch.Teardown); err != nil {
		fmt.Fprintf(os.Stderr, "warning: chapter %s teardown failed: %v\n", ch.ID, err)
	}
	dctx, cancel := context.WithTimeout(ctx, engine.ChapterDeployTimeout(ch))
	defer cancel()
	topo, err := engine.PrepareTopology(ch)
	if err != nil {
		return err
	}
	res, err := a.eng.Destroy(dctx, topo)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab destroy failed (code=%d): %s", res.Code, res.Stderr)
	}
	return nil
}

// makeVerifier combines assertions and the C build into a watch.Verifier. File
// assertions run first (sentinel gate); if any fails, the rest are skipped.
// For code-implement exercises the binary is rebuilt between the file gate and
// the remaining assertions.
func makeVerifier(eng *engine.Engine, builder reset.Rebuilder) func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
	return func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
		var topo *engine.Topology
		if ex.ChapterRef != nil {
			t, err := engine.ParseTopology(ex.ChapterRef.TopologyPath)
			if err != nil {
				return nil, fmt.Errorf("parse topology %s: %w", ex.ChapterRef.TopologyPath, err)
			}
			topo = t
		}
		fileResults := assert.RunFiltered(ctx, eng, topo, ex, func(a content.Assertion) bool {
			return a.Type == "file"
		})
		for _, r := range fileResults {
			if !r.Passed {
				return fileResults, nil
			}
		}
		if ex.Type == content.TypeCodeImplement {
			if builder == nil {
				fail := assert.Result{
					Exercise: ex.FullID(), Name: "build, deploy and start", Type: "build",
					Layer: "data-plane", Expected: "compile, deploy and start succeed",
					Actual: "no rebuilder configured", Passed: false,
				}
				return append(fileResults, fail), nil
			}
			if err := builder.Rebuild(ctx, ex); err != nil {
				fail := assert.Result{
					Exercise: ex.FullID(), Name: "build, deploy and start", Type: "build",
					Layer: "data-plane", Expected: "compile, deploy and start succeed",
					Actual: "failure", Err: err, Passed: false,
				}
				return append(fileResults, fail), nil
			}
		}
		rest := assert.RunFiltered(ctx, eng, topo, ex, func(a content.Assertion) bool {
			return a.Type != "file"
		})
		return append(fileResults, rest...), nil
	}
}
