// Package reset implements two-tier per-exercise reset.
package reset

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
)

// Engine is the minimal engine capability needed for fast-path reset.
type Engine interface {
	Redeploy(ctx context.Context, ch *content.Chapter) error
	CopyToContainer(ctx context.Context, src, container, dest string) error
	Exec(ctx context.Context, node, cmd string) error
}

// Rebuilder is the extension point for code-implement fast-path reset (implemented in P5).
type Rebuilder interface {
	Rebuild(ctx context.Context, ex *content.Exercise) error
}

// GitCheckout restores tracked files in the host working tree.
type GitCheckout func(ctx context.Context, root string, paths []string) error

// Options configures a Resetter.
type Options struct {
	Root      string
	Engine    Engine
	Git       GitCheckout
	Rebuilder Rebuilder // may be nil
}

// Resetter performs fast/hard reset.
type Resetter struct {
	opts Options
}

// New builds a Resetter; a nil Git uses the default git implementation.
func New(opts Options) *Resetter {
	if opts.Git == nil {
		opts.Git = DefaultGitCheckout
	}
	return &Resetter{opts: opts}
}

// Fast runs the fast path: git checkout -> docker cp -> reload; code exercises rebuild.
func (r *Resetter) Fast(ctx context.Context, ex *content.Exercise) error {
	ctx, cancel := context.WithTimeout(ctx, exerciseTimeout(ex))
	defer cancel()
	if err := r.checkoutFiles(ctx, ex); err != nil {
		return err
	}
	if ex.Type == content.TypeCodeImplement {
		if r.opts.Rebuilder == nil {
			return fmt.Errorf("exercise %s is code-implement but no rebuilder is configured", ex.FullID())
		}
		if err := r.opts.Rebuilder.Rebuild(ctx, ex); err != nil {
			return err
		}
		return nil
	}
	if err := r.seedFiles(ctx, ex); err != nil {
		return err
	}
	return r.runReload(ctx, ex)
}

// Hard runs the hard path: redeploy the chapter topology and re-seed.
func (r *Resetter) Hard(ctx context.Context, ex *content.Exercise) error {
	ctx, cancel := context.WithTimeout(ctx, exerciseTimeout(ex)+2*time.Minute)
	defer cancel()
	if ex.ChapterRef == nil {
		return fmt.Errorf("exercise %s has no chapter reference", ex.FullID())
	}
	if err := r.opts.Engine.Redeploy(ctx, ex.ChapterRef); err != nil {
		return err
	}
	if err := r.checkoutFiles(ctx, ex); err != nil {
		return err
	}
	if ex.Type == content.TypeCodeImplement {
		if r.opts.Rebuilder == nil {
			return fmt.Errorf("exercise %s is code-implement but no rebuilder is configured", ex.FullID())
		}
		return r.opts.Rebuilder.Rebuild(ctx, ex)
	}
	if err := r.seedFiles(ctx, ex); err != nil {
		return err
	}
	return r.runReload(ctx, ex)
}

func (r *Resetter) checkoutFiles(ctx context.Context, ex *content.Exercise) error {
	var rel []string
	for _, f := range ex.Files {
		rel = append(rel, filepath.Join(relDir(r.opts.Root, ex.Dir), f.Source))
	}
	if len(rel) == 0 {
		return nil
	}
	return r.opts.Git(ctx, r.opts.Root, rel)
}

func (r *Resetter) seedFiles(ctx context.Context, ex *content.Exercise) error {
	for _, f := range ex.Files {
		if f.Node == "" || f.Dest == "" {
			continue
		}
		src := filepath.Join(ex.Dir, f.Source)
		container := contentContainer(ex, f.Node)
		if err := r.opts.Engine.CopyToContainer(ctx, src, container, f.Dest); err != nil {
			return fmt.Errorf("seed %s -> %s:%s: %w", f.Source, f.Node, f.Dest, err)
		}
	}
	return nil
}

func (r *Resetter) runReload(ctx context.Context, ex *content.Exercise) error {
	for _, c := range ex.Reload {
		if err := r.opts.Engine.Exec(ctx, contentContainer(ex, c.Node), c.Cmd); err != nil {
			return fmt.Errorf("reload on %s failed: %w", c.Node, err)
		}
	}
	return nil
}

func relDir(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return rel
}

// exerciseTimeout returns the exercise's declared timeout, default 30s.
func exerciseTimeout(ex *content.Exercise) time.Duration {
	if d, err := time.ParseDuration(ex.Timeout); err == nil && d > 0 {
		return d
	}
	return 30 * time.Second
}

// contentContainer derives the container long name from the chapter topology name; falls back to the node name.
func contentContainer(ex *content.Exercise, node string) string {
	if ex.ChapterRef != nil && ex.ChapterRef.TopologyName != "" {
		return "clab-" + ex.ChapterRef.TopologyName + "-" + node
	}
	return node
}
