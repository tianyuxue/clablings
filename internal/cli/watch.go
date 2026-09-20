package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
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
			return runQuiz(qctx, byteCh, out, ex, loc)
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
		Report: func(r []assert.Result) {
			clearStatus(out)
			report.Text(out, r)
		},
		Status: func(m string) { fmt.Fprintln(out, m) },
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
		return watchLoop(ctx, sess, lib, cfg, loc, byteCh, ticker.C, out, errOut)
	}
	return watchLoop(ctx, sess, lib, cfg, loc, nil, ticker.C, out, errOut)
}

func watchLoop(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, loc string, byteCh <-chan byte, tick <-chan time.Time, out, errOut io.Writer) error {
	var w *watch.Watcher
	shown := ""
	// refresh prints the context block once per exercise (progress, brief,
	// legend) and prepares a watcher for it. Actions never reprint it.
	refresh := func() {
		cur := sess.Current()
		if cur == nil || cur.FullID() == shown {
			return
		}
		printContext(sess, lib, cur, out)
		shown = cur.FullID()
		w = nil
		// Only exercises gated on a file assertion react to file edits; runtime
		// exercises (config-fix) are verified explicitly with Enter.
		if usesFileTrigger(cur) {
			w = watch.NewWatcher(cur, cfg.Watch.Debounce)
			if _, err := w.Scan(); err != nil {
				fmt.Fprintf(errOut, "warning: %v\n", err)
			}
		}
	}
	after := func(done bool, err error) (bool, error) {
		if err != nil {
			return true, err
		}
		if done || sess.Current() == nil {
			return true, nil
		}
		if quit, clean := drainKeys(byteCh); quit || clean {
			return true, handleQuit(ctx, sess, clean, out, errOut)
		}
		refresh()
		return false, nil
	}
	refresh()
	for {
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
				done, err := verifyAndReload(ctx, sess, lib, cfg, out, errOut)
				if err != nil {
					return err
				}
				stop, qerr := after(done, nil)
				if stop {
					return qerr
				}
			case watch.KeyHint:
				watch.WriteHints(out, sess.Current())
			case watch.KeyConcept:
				cur := sess.Current()
				if cur == nil || cur.ChapterRef == nil {
					fmt.Fprintln(out, "no concept for this exercise")
				} else if err := printChapterRead(out, cur.ChapterRef, loc); err != nil {
					fmt.Fprintf(errOut, "read error: %v\n", err)
				}
			case watch.KeyNext, watch.KeyPrev:
				delta := 1
				if k == watch.KeyPrev {
					delta = -1
				}
				ok, err := sess.Step(ctx, delta)
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(out, "no exercise in that direction")
					continue
				}
				refresh()
			case watch.KeyRedeploy:
				if err := sess.ResetCurrent(ctx, true); err != nil {
					fmt.Fprintf(errOut, "reset error: %v\n", err)
				}
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
			case watch.KeyFastReset:
				if err := sess.ResetCurrent(ctx, false); err != nil {
					fmt.Fprintf(errOut, "reset error: %v\n", err)
				}
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
				}
			case watch.KeyQuit:
				return handleQuit(ctx, sess, false, out, errOut)
			case watch.KeyCleanQuit:
				return handleQuit(ctx, sess, true, out, errOut)
			}
		case <-tick:
			if w == nil {
				continue
			}
			if _, err := w.Scan(); err != nil {
				continue
			}
			if w.ReadyToVerify() {
				w.Consume()
				done, err := verifyAndReload(ctx, sess, lib, cfg, out, errOut)
				if err != nil {
					return err
				}
				stop, qerr := after(done, nil)
				if stop {
					return qerr
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

// verifyAndReload verifies the current exercise. It returns done=true when the
// whole library is complete or the user asked to stop.
func verifyAndReload(ctx context.Context, sess *watch.Session, lib *content.Library, cfg *config.Config, out, errOut io.Writer) (bool, error) {
	wasQuiz := sess.Current() != nil && sess.Current().Type == content.TypeQuiz
	if !wasQuiz {
		writeStatus(out, yellow("→ verifying ..."))
	}
	if err := sess.VerifyNow(ctx); err != nil {
		clearStatus(out)
		if errors.Is(err, watch.ErrQuizAborted) {
			fmt.Fprintln(out, "quiz aborted")
			fmt.Fprintln(out, quitNotice)
			return true, nil
		}
		if wasQuiz && errors.Is(err, context.Canceled) {
			fmt.Fprintln(out, "quiz aborted")
			fmt.Fprintln(out, quitNotice)
			return true, nil
		}
		if errors.Is(err, watch.ErrQuizUnsupported) {
			return false, err
		}
		if errors.Is(err, watch.ErrQuizNext) || errors.Is(err, watch.ErrQuizPrev) {
			delta := 1
			if errors.Is(err, watch.ErrQuizPrev) {
				delta = -1
			}
			ok, serr := sess.Step(ctx, delta)
			if serr != nil {
				return false, serr
			}
			if !ok {
				fmt.Fprintln(out, "no exercise in that direction")
			}
			return false, nil
		}
		fmt.Fprintf(errOut, "verify error: %v\n", err)
		return false, nil
	}
	if sess.Current() == nil {
		fmt.Fprintln(out, green("all exercises complete"))
		return true, nil
	}
	return false, nil
}

// printContext renders the once-per-exercise header: progress bar, title,
// brief, key legend, and a quiz start hint.
func printContext(sess *watch.Session, lib *content.Library, ex *content.Exercise, out io.Writer) {
	idx, total := position(lib, ex)
	fmt.Fprintf(out, "%s %s\n", bold("Progress:"), progressBar(idx, total))
	fmt.Fprintf(out, "%s\n", bold(fmt.Sprintf("%s: %s", ex.ID, ex.Title)))
	if ex.Brief != "" {
		_ = renderMarkdown(out, ex.Brief+"\n")
	}
	watch.WriteLegend(out)
	if ex.Type == content.TypeQuiz {
		fmt.Fprintln(out, dim("Press Enter to start the quiz."))
	}
	fmt.Fprintln(out)
}

// position returns the 1-based index of ex within the library and its total.
func position(lib *content.Library, ex *content.Exercise) (int, int) {
	total := len(lib.Order)
	for i, e := range lib.Order {
		if e.FullID() == ex.FullID() {
			return i + 1, total
		}
	}
	return 0, total
}

// usesFileTrigger reports whether an exercise should auto-verify when its
// watched files change. Only exercises with a file assertion (the sentinel
// gate) need it; runtime exercises are verified explicitly.
func usesFileTrigger(ex *content.Exercise) bool {
	for _, a := range ex.Assertions {
		if a.Type == "file" {
			return true
		}
	}
	return false
}
