package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/config"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/report"
	"github.com/tianyuxue/clablings/internal/reset"
	"github.com/tianyuxue/clablings/internal/state"
	"github.com/tianyuxue/clablings/internal/watch"
)

func newWatchCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Run the watch loop: react to file changes and Enter, verify and advance",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			return runWatch(cmd.Context(), resolved, "")
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	return cmd
}

func runWatch(ctx context.Context, root, initialID string) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: reading config failed, using defaults: %v\n", err)
	}
	clabBin := binOr(cfg.Containerlab.BinPath, "containerlab")
	dockerBin := binOr(cfg.Docker.BinPath, "docker")

	lib, err := content.Load(root)
	if err != nil {
		return err
	}
	st, err := state.Load(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v\n", err)
	}
	eng := engine.New(clabBin, dockerBin)
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng)})
	sess := watch.New(watch.Options{
		Library: lib,
		State:   st,
		Engine:  newLifecycleAdapter(eng),
		Reset:   res,
		Verify:  makeVerifier(eng),
		Report:  func(r []assert.Result) { report.Text(os.Stdout, r) },
	})
	if initialID != "" {
		if err := sess.Run(ctx, initialID); err != nil {
			return err
		}
	} else if err := sess.Start(ctx); err != nil {
		return err
	}
	printProgress(sess, lib)

	ticker := time.NewTicker(cfg.Watch.PollInterval)
	defer ticker.Stop()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if raw := watch.MakeRaw(int(os.Stdin.Fd())); raw != nil {
			defer term.Restore(int(os.Stdin.Fd()), raw)
			enterCh := make(chan struct{}, 1)
			go watch.ReadEnter(os.Stdin, enterCh)
			return watchLoop(ctx, sess, lib, cfg, enterCh, ticker.C)
		}
	}
	return watchLoop(ctx, sess, lib, cfg, nil, ticker.C)
}

func watchLoop(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, enterCh <-chan struct{}, tick <-chan time.Time) error {
	w := watch.NewWatcher(sess.Current(), cfg.Watch.Debounce)
	if _, err := w.Scan(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-enterCh:
			done, err := verifyAndReload(ctx, sess, lib, &w, cfg)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		case <-tick:
			if _, err := w.Scan(); err != nil {
				continue
			}
			if w.ReadyToVerify() {
				w.Consume()
				done, err := verifyAndReload(ctx, sess, lib, &w, cfg)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
			}
		}
	}
}

// verifyAndReload verifies the current exercise and, unless the library is
// complete, resets the watcher for the (possibly new) current exercise.
func verifyAndReload(ctx context.Context, sess *watch.Session, lib *content.Library, w **watch.Watcher, cfg *config.Config) (bool, error) {
	if err := sess.VerifyNow(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "verify error: %v\n", err)
		return false, nil
	}
	printProgress(sess, lib)
	if sess.Current() == nil {
		return true, nil
	}
	*w = watch.NewWatcher(sess.Current(), cfg.Watch.Debounce)
	_, _ = (*w).Scan()
	return false, nil
}

func printProgress(sess *watch.Session, lib *content.Library) {
	cur := sess.Current()
	if cur == nil {
		fmt.Println("all exercises complete")
		return
	}
	total := len(lib.Order)
	idx := 0
	for i, ex := range lib.Order {
		if ex.FullID() == cur.FullID() {
			idx = i + 1
			break
		}
	}
	name := filepath.Base(cur.Dir)
	fmt.Printf("[%d/%d] %s: In Progress\n", idx, total, name)
}
