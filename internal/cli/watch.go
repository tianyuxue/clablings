package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/build"
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
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(errOut, "warning: reading config failed, using defaults: %v\n", err)
	}
	clabBin := binOr(cfg.Containerlab.BinPath, "containerlab")
	dockerBin := binOr(cfg.Docker.BinPath, "docker")

	loc := locale(errOut)
	lib, err := content.LoadWithLocale(root, loc)
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

	rawActive := false
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if raw := watch.MakeRaw(int(os.Stdin.Fd())); raw != nil {
			defer term.Restore(int(os.Stdin.Fd()), raw)
			rawActive = true
		}
	}
	if rawActive {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			out = watch.NewCRLFWriter(out)
		}
		if term.IsTerminal(int(os.Stderr.Fd())) {
			errOut = watch.NewCRLFWriter(errOut)
		}
	}

	var byteCh chan byte
	var quizHandler func(context.Context, *content.Exercise) ([]assert.Result, error)
	if rawActive {
		byteCh = make(chan byte, 256)
		go watch.ReadBytes(os.Stdin, byteCh)
		quizHandler = func(qctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
			return nil, fmt.Errorf("%w: exercise %s (use `clablings test --answers`)", watch.ErrQuizUnsupported, ex.FullID())
		}
	}

	eng := engine.New(clabBin, dockerBin)
	builder := build.New(eng.Runner(), dockerBin, cfg.Build.DefaultImage)
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng), Rebuilder: builder})
	sess := watch.New(watch.Options{
		Library: lib,
		State:   st,
		Engine:  newLifecycleAdapter(eng),
		Reset:   res,
		Verify:  makeVerifier(eng, builder),
		Quiz:    quizHandler,
		Report:  func(r []assert.Result) { report.Text(out, r) },
		Status:  func(m string) { fmt.Fprintln(out, m) },
	})
	if initialID != "" {
		if err := sess.Run(ctx, initialID); err != nil {
			return err
		}
	} else if err := sess.Start(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(cfg.Watch.PollInterval)
	defer ticker.Stop()

	if rawActive {
		printProgress(sess, lib, out)
		watch.WriteLegend(out)
		return watchLoop(ctx, sess, lib, cfg, loc, byteCh, ticker.C, out, errOut)
	}
	printProgress(sess, lib, out)
	return watchLoop(ctx, sess, lib, cfg, loc, nil, ticker.C, out, errOut)
}

func watchLoop(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, loc string, byteCh <-chan byte, tick <-chan time.Time, out, errOut io.Writer) error {
	interactive := byteCh != nil
	w := watch.NewWatcher(sess.Current(), cfg.Watch.Debounce)
	if _, err := w.Scan(); err != nil {
		fmt.Fprintf(errOut, "warning: %v\n", err)
	}
	for {
		if cur := sess.Current(); cur != nil && cur.Type == content.TypeQuiz {
			if ctx.Err() != nil {
				return handleQuit(ctx, sess, false, out, errOut)
			}
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
			if sess.Current() == cur {
				return fmt.Errorf("quiz %s was not completed", cur.FullID())
			}
			if interactive {
				watch.WriteLegend(out)
			}
			continue
		}
		select {
		case <-ctx.Done():
			return handleQuit(ctx, sess, false, out, errOut)
		case b, alive := <-byteCh:
			if !alive {
				byteCh = nil
				continue
			}
			k, ok := watch.KeyFor(b)
			if !ok {
				continue
			}
			switch k {
			case watch.KeyEnter:
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
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
				if interactive {
					watch.WriteLegend(out)
				}
			case watch.KeyHint:
				watch.WriteHints(out, sess.Current())
				if interactive {
					watch.WriteLegend(out)
				}
			case watch.KeyConcept:
				cur := sess.Current()
				if cur == nil || cur.ChapterRef == nil {
					fmt.Fprintln(out, "no concept for this exercise")
				} else if err := printChapterRead(out, cur.ChapterRef, loc); err != nil {
					fmt.Fprintf(errOut, "read error: %v\n", err)
				}
				if interactive {
					watch.WriteLegend(out)
				}
			case watch.KeyRedeploy:
				if err := sess.ResetCurrent(ctx, true); err != nil {
					fmt.Fprintf(errOut, "reset error: %v\n", err)
				}
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
				if interactive {
					watch.WriteLegend(out)
				}
			case watch.KeyFastReset:
				if err := sess.ResetCurrent(ctx, false); err != nil {
					fmt.Fprintf(errOut, "reset error: %v\n", err)
				}
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
				if interactive {
					watch.WriteLegend(out)
				}
			case watch.KeyQuit:
				return handleQuit(ctx, sess, false, out, errOut)
			case watch.KeyCleanQuit:
				return handleQuit(ctx, sess, true, out, errOut)
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
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
				if interactive {
					watch.WriteLegend(out)
				}
			}
		}
	}
}

const quitNotice = "containers kept running; press x to clean up"

// handleQuit implements the two exit paths: keep containers (q/Ctrl+C) or
// remove every chapter first (x).
func handleQuit(ctx context.Context, sess *watch.Session, clean bool, out, errOut io.Writer) error {
	if !clean {
		fmt.Fprintln(out, quitNotice)
		return nil
	}
	if err := sess.Cleanup(ctx); err != nil {
		fmt.Fprintf(errOut, "cleanup error: %v\n", err)
		return err
	}
	fmt.Fprintln(out, "containers removed")
	return nil
}

// drainKeys discards input buffered during a blocking action, reporting whether
// a quit or clean-quit was requested among buffered bytes.
func drainKeys(ch <-chan byte) (quit, cleanQuit bool) {
	for {
		select {
		case b, ok := <-ch:
			if !ok {
				return
			}
			k, mapped := watch.KeyFor(b)
			if !mapped {
				continue
			}
			switch k {
			case watch.KeyQuit:
				quit = true
			case watch.KeyCleanQuit:
				cleanQuit = true
			}
		default:
			return
		}
	}
}

// verifyAndReload verifies the current exercise and, unless the library is
// complete, returns a fresh watcher for the (possibly new) current exercise.
// On verify failure it returns (nil, false, nil) so the caller keeps using the
// existing watcher.
func verifyAndReload(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, out, errOut io.Writer) (*watch.Watcher, bool, error) {
	if err := sess.VerifyNow(ctx); err != nil {
		if errors.Is(err, watch.ErrQuizAborted) {
			fmt.Fprintln(out, "quiz aborted")
			return nil, true, nil
		}
		if errors.Is(err, watch.ErrQuizUnsupported) {
			return nil, false, err
		}
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
