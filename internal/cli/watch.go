package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
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
			return runWatch(cmd.Context(), resolved, "", cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	return cmd
}

func runWatch(ctx context.Context, root, initialID string, out, errOut io.Writer) error {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(errOut, "warning: reading config failed, using defaults: %v\n", err)
	}
	clabBin := binOr(cfg.Containerlab.BinPath, "containerlab")
	dockerBin := binOr(cfg.Docker.BinPath, "docker")

	lib, err := content.Load(root)
	if err != nil {
		return err
	}
	st, err := state.Load(root)
	if err != nil {
		var corrupt *state.CorruptError
		if !errors.As(err, &corrupt) {
			return err
		}
		fmt.Fprintf(errOut, "warning: %v\n", err)
	}
	eng := engine.New(clabBin, dockerBin)
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng)})
	sess := watch.New(watch.Options{
		Library: lib,
		State:   st,
		Engine:  newLifecycleAdapter(eng),
		Reset:   res,
		Verify:  makeVerifier(eng),
		Report:  func(r []assert.Result) { report.Text(out, r) },
	})
	if initialID != "" {
		if err := sess.Run(ctx, initialID); err != nil {
			return err
		}
	} else if err := sess.Start(ctx); err != nil {
		return err
	}
	printProgress(sess, lib, out)

	ticker := time.NewTicker(cfg.Watch.PollInterval)
	defer ticker.Stop()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if raw := watch.MakeRaw(int(os.Stdin.Fd())); raw != nil {
			defer term.Restore(int(os.Stdin.Fd()), raw)
			enterCh := make(chan struct{}, 1)
			go watch.ReadEnter(os.Stdin, enterCh)
			return watchLoop(ctx, sess, lib, cfg, enterCh, ticker.C, out, errOut)
		}
	}
	return watchLoop(ctx, sess, lib, cfg, nil, ticker.C, out, errOut)
}

func watchLoop(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, enterCh <-chan struct{}, tick <-chan time.Time, out, errOut io.Writer) error {
	w := watch.NewWatcher(sess.Current(), cfg.Watch.Debounce)
	if _, err := w.Scan(); err != nil {
		fmt.Fprintf(errOut, "warning: %v\n", err)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-enterCh:
			next, done, err := verifyAndReload(ctx, sess, lib, cfg, out, errOut)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
			if next != nil {
				w = next
			}
		case <-tick:
			if _, err := w.Scan(); err != nil {
				continue
			}
			if w.ReadyToVerify() {
				w.Consume()
				next, done, err := verifyAndReload(ctx, sess, lib, cfg, out, errOut)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
				if next != nil {
					w = next
				}
			}
		}
	}
}

// verifyAndReload verifies the current exercise and, unless the library is
// complete, returns a fresh watcher for the (possibly new) current exercise.
// On verify failure it returns (nil, false, nil) so the caller keeps using the
// existing watcher.
func verifyAndReload(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, out, errOut io.Writer) (*watch.Watcher, bool, error) {
	if err := sess.VerifyNow(ctx); err != nil {
		fmt.Fprintf(errOut, "verify error: %v\n", err)
		return nil, false, nil
	}
	printProgress(sess, lib, out)
	if sess.Current() == nil {
		return nil, true, nil
	}
	nw := watch.NewWatcher(sess.Current(), cfg.Watch.Debounce)
	if _, err := nw.Scan(); err != nil {
		fmt.Fprintf(errOut, "warning: %v\n", err)
	}
	return nw, false, nil
}

func printProgress(sess *watch.Session, lib *content.Library, out io.Writer) {
	cur := sess.Current()
	if cur == nil {
		fmt.Fprintln(out, "all exercises complete")
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
	fmt.Fprintf(out, "[%d/%d] %s: In Progress\n", idx, total, name)
}
