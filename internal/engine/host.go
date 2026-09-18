package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
)

// RunHostCommands runs host-side commands in order, stopping at the first
// failure. Context cancellation is surfaced as ctx.Err(); WaitDelay bounds the
// wait for orphaned descendants holding inherited pipes.
func RunHostCommands(ctx context.Context, cmds []string) error {
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

// ChapterDeployTimeout returns the chapter's declared deploy timeout, default 2m.
func ChapterDeployTimeout(ch *content.Chapter) time.Duration {
	if d, err := time.ParseDuration(ch.Deploy.Timeout); err == nil && d > 0 {
		return d
	}
	return 2 * time.Minute
}
