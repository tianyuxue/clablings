package reset

import (
	"context"
	"fmt"
	"os"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

// FromEngine adapts a real engine.Engine to reset.Engine.
func FromEngine(e *engine.Engine) Engine { return &engineAdapter{e: e} }

type engineAdapter struct{ e *engine.Engine }

func (a *engineAdapter) Redeploy(ctx context.Context, ch *content.Chapter) error {
	if !ch.HasTopology() {
		return nil
	}
	if err := engine.RunHostCommands(ctx, ch.Teardown); err != nil {
		fmt.Fprintf(os.Stderr, "warning: chapter %s teardown failed: %v\n", ch.ID, err)
	}
	topo, err := engine.PrepareTopology(ch)
	if err != nil {
		return fmt.Errorf("chapter %s: %w", ch.ID, err)
	}
	rctx, cancel := context.WithTimeout(ctx, engine.ChapterDeployTimeout(ch))
	defer cancel()
	res, err := a.e.Redeploy(rctx, topo, engine.ChapterDeployTimeout(ch))
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab redeploy failed (code=%d): %s", res.Code, res.Stderr)
	}
	if err := engine.RunHostCommands(rctx, ch.Setup); err != nil {
		return fmt.Errorf("chapter %s setup failed: %w", ch.ID, err)
	}
	return nil
}

func (a *engineAdapter) CopyToContainer(ctx context.Context, src, container, dest string) error {
	return a.e.CopyToContainer(ctx, src, container, dest)
}

func (a *engineAdapter) Exec(ctx context.Context, node, cmd string) error {
	res, err := a.e.Exec(ctx, node, cmd)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("command on %s failed (code=%d): %s", node, res.Code, res.Stderr)
	}
	return nil
}
