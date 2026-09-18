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
		return "", err
	}
	if res.Code != 0 {
		return "", fmt.Errorf("build failed (code=%d):\n%s%s", res.Code, res.Stdout, res.Stderr)
	}
	out := filepath.Join(ex.Dir, ex.Build.Output)
	if _, err := os.Stat(out); err != nil {
		return "", fmt.Errorf("build succeeded but artifact %s not found: %w", ex.Build.Output, err)
	}
	return out, nil
}

// exerciseTimeout returns the exercise's declared timeout, default 30s.
func exerciseTimeout(ex *content.Exercise) time.Duration {
	if d, err := time.ParseDuration(ex.Timeout); err == nil && d > 0 {
		return d
	}
	return 30 * time.Second
}
