// Package build implements in-container compilation, binary deployment and
// process restart for code-implement exercises.
package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

// Builder builds and manages an exercise's binary via docker.
type Builder struct {
	runner       engine.Runner
	dockerBin    string
	defaultImage string
}

// New builds a Builder.
func New(runner engine.Runner, dockerBin, defaultImage string) *Builder {
	return &Builder{runner: runner, dockerBin: dockerBin, defaultImage: defaultImage}
}

// Build runs the exercise's build command in a container and returns the
// absolute path of the output artifact.
func (b *Builder) Build(ctx context.Context, ex *content.Exercise) (string, error) {
	if ex.Build == nil {
		return "", fmt.Errorf("exercise %s has no build spec", ex.FullID())
	}
	ctx, cancel := context.WithTimeout(ctx, exerciseTimeout(ex))
	defer cancel()
	image := ex.Build.Image
	if image == "" {
		image = b.defaultImage
	}
	user := fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	res, err := b.runner.Run(ctx, b.dockerBin, "run", "--rm",
		"--user", user,
		"-v", ex.Dir+":/src", "-w", "/src",
		image, "sh", "-lc", ex.Build.Command)
	if err != nil {
		return "", fmt.Errorf("build %s failed to start: %w", ex.FullID(), err)
	}
	if res.Code != 0 {
		return "", fmt.Errorf("build failed (code=%d):\n%s\n%s", res.Code, res.Stdout, res.Stderr)
	}
	out := filepath.Join(ex.Dir, ex.Build.Output)
	st, err := os.Stat(out)
	if err != nil {
		return "", fmt.Errorf("build succeeded but artifact %s not found: %w", ex.Build.Output, err)
	}
	if st.IsDir() {
		return "", fmt.Errorf("build output %s is a directory, expected a file", ex.Build.Output)
	}
	return out, nil
}

// Deploy copies the binary into the target node.
func (b *Builder) Deploy(ctx context.Context, ex *content.Exercise, bin string) error {
	if ex.Run == nil {
		return fmt.Errorf("exercise %s has no run spec", ex.FullID())
	}
	ctx, cancel := context.WithTimeout(ctx, exerciseTimeout(ex))
	defer cancel()
	container := containerName(ex, ex.Run.Node)
	res, err := b.runner.Run(ctx, b.dockerBin, "cp", bin, container+":"+ex.Run.Dest)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("binary deploy failed (code=%d): %s", res.Code, res.Stderr)
	}
	return nil
}

// Restart stops and restarts the process inside the node. The stop exit code is
// tolerated; the start must succeed.
func (b *Builder) Restart(ctx context.Context, ex *content.Exercise) error {
	if ex.Run == nil {
		return fmt.Errorf("exercise %s has no run spec", ex.FullID())
	}
	ctx, cancel := context.WithTimeout(ctx, exerciseTimeout(ex))
	defer cancel()
	container := containerName(ex, ex.Run.Node)
	if ex.Run.Stop != "" {
		if _, err := b.runner.Run(ctx, b.dockerBin, "exec", container, "sh", "-lc", ex.Run.Stop); err != nil {
			return err
		}
	}
	res, err := b.runner.Run(ctx, b.dockerBin, "exec", container, "sh", "-lc", ex.Run.Start)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("process start failed (code=%d): %s", res.Code, res.Stderr)
	}
	return nil
}

// Rebuild implements reset.Rebuilder: build -> deploy -> restart.
func (b *Builder) Rebuild(ctx context.Context, ex *content.Exercise) error {
	bin, err := b.Build(ctx, ex)
	if err != nil {
		return err
	}
	if err := b.Deploy(ctx, ex, bin); err != nil {
		return err
	}
	return b.Restart(ctx, ex)
}

func containerName(ex *content.Exercise, node string) string {
	if ex.ChapterRef != nil && ex.ChapterRef.TopologyName != "" {
		return "clab-" + ex.ChapterRef.TopologyName + "-" + node
	}
	return node
}

// exerciseTimeout returns the exercise's declared timeout, default 30s.
func exerciseTimeout(ex *content.Exercise) time.Duration {
	if d, err := time.ParseDuration(ex.Timeout); err == nil && d > 0 {
		return d
	}
	return 30 * time.Second
}
