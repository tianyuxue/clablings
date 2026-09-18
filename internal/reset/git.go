package reset

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DefaultGitCheckout discards working-tree edits to tracked files (restores
// them from the index). It surfaces git's output on failure.
func DefaultGitCheckout(ctx context.Context, root string, paths []string) error {
	args := append([]string{"-C", root, "checkout", "--"}, paths...)
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed (try --hard or verify the library repo): %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
