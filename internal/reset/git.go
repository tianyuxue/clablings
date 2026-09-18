package reset

import (
	"context"
	"os/exec"
)

// DefaultGitCheckout restores tracked files to their working-tree version.
func DefaultGitCheckout(ctx context.Context, root string, paths []string) error {
	args := append([]string{"-C", root, "checkout", "--"}, paths...)
	return exec.CommandContext(ctx, "git", args...).Run()
}
