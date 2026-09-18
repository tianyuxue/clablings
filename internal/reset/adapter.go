package reset

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

// FromEngine adapts a real engine.Engine to reset.Engine.
func FromEngine(e *engine.Engine) Engine { return &engineAdapter{e: e} }

type engineAdapter struct{ e *engine.Engine }

func (a *engineAdapter) Redeploy(ctx context.Context, ch *content.Chapter) error {
	if err := runHostCommands(ctx, ch.Teardown); err != nil {
		fmt.Fprintf(os.Stderr, "warning: chapter %s teardown failed: %v\n", ch.ID, err)
	}
	rctx, cancel := context.WithTimeout(ctx, deployTimeout(ch))
	defer cancel()
	res, err := a.e.Redeploy(rctx, ch.TopologyPath, deployTimeout(ch))
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("containerlab redeploy failed (code=%d): %s", res.Code, res.Stderr)
	}
	if err := runHostCommands(rctx, ch.Setup); err != nil {
		return fmt.Errorf("chapter %s setup failed: %w", ch.ID, err)
	}
	return nil
}

// runHostCommands runs host-side commands in order, stopping at the first failure.
func runHostCommands(ctx context.Context, cmds []string) error {
	for _, c := range cmds {
		cmd := exec.CommandContext(ctx, "sh", "-lc", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%q: %w", c, err)
		}
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

func deployTimeout(ch *content.Chapter) time.Duration {
	if d, err := time.ParseDuration(ch.Deploy.Timeout); err == nil && d > 0 {
		return d
	}
	return 2 * time.Minute
}
