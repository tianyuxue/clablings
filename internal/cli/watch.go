package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
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
	// forceRefresh reprints the context block even when the exercise has not
	// changed; used after the picker closes without jumping so the screen is not
	// left blank.
	forceRefresh := func() {
		shown = ""
		refresh()
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

	// Verification runs in the background so the key loop stays responsive;
	// Esc cancels the in-flight verification.
	type outcome struct {
		done bool
		err  error
	}
	var vch chan outcome
	var vcancel context.CancelFunc
	pendingQuit, pendingClean := false, false
	startVerify := func() {
		if vch != nil {
			return
		}
		writeStatus(out, yellow("→ verifying ...")+dim("  (Esc to cancel)"))
		vctx, cancel := context.WithCancel(ctx)
		vcancel = cancel
		pendingQuit, pendingClean = false, false
		vch = make(chan outcome, 1)
		go func() {
			done, err := verifyAndReload(vctx, sess, lib, cfg, out, errOut)
			vch <- outcome{done: done, err: err}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			if vcancel != nil {
				vcancel()
			}
			return handleQuit(ctx, sess, false, out, errOut)
		case res := <-vch:
			vch = nil
			vcancel = nil
			if pendingQuit || pendingClean {
				if res.err != nil {
					return res.err
				}
				return handleQuit(ctx, sess, pendingClean, out, errOut)
			}
			stop, qerr := after(res.done, res.err)
			if stop {
				return qerr
			}
		case b, alive := <-byteCh:
			if !alive {
				byteCh = nil
				continue
			}
			if vch != nil {
				// While verifying, Esc cancels; q/x cancel and then quit. A
				// leading ESC may also begin an ANSI sequence (arrow keys),
				// which is consumed and ignored rather than treated as cancel.
				if b == 0x1b {
					follow, seq := classifyEsc(byteCh)
					if seq {
						continue
					}
					vcancel()
					if follow < 0 {
						continue
					}
					b = byte(follow)
				}
				switch {
				case b == 'q' || b == 'Q' || b == 0x03:
					pendingQuit = true
					vcancel()
				case b == 'x' || b == 'X':
					pendingClean = true
					vcancel()
				}
				continue
			}
			k, ok := watch.KeyFor(b)
			if !ok {
				continue
			}
			switch k {
			case watch.KeyEnter:
				startVerify()
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
			case watch.KeyList:
				cur := sess.Current()
				if cur == nil {
					continue
				}
				id, err := runPicker(ctx, byteCh, out, lib, cur.FullID(), sess.StatusOf)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						forceRefresh()
						continue
					}
					return err
				}
				if id == "" || id == cur.FullID() {
					forceRefresh()
					continue
				}
				if err := sess.Run(ctx, id); err != nil {
					fmt.Fprintf(errOut, "jump error: %v\n", err)
					forceRefresh()
					continue
				}
				if quit, clean := drainKeys(byteCh); quit || clean {
					return handleQuit(ctx, sess, clean, out, errOut)
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
			if vch != nil || w == nil {
				continue
			}
			if _, err := w.Scan(); err != nil {
				continue
			}
			if w.ReadyToVerify() {
				w.Consume()
				startVerify()
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
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(out, yellow("verification canceled"))
			return false, nil
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

// printContext renders the once-per-exercise header: progress bar, highlighted
// title, meta line, task brief, key legend, and a quiz start hint.
func printContext(sess *watch.Session, lib *content.Library, ex *content.Exercise, out io.Writer) {
	idx, total := position(lib, ex)
	fmt.Fprintf(out, "\n%s %s\n", bold("Progress:"), progressBar(idx, total))
	fmt.Fprintf(out, " %s %s %s %s\n",
		cyan("▌"), bold(cyan(ex.ID)), dim("·"), bold(ex.Title))
	fmt.Fprintf(out, "   %s %s\n", magenta(string(ex.Type)), stars(ex.Difficulty))
	if ex.Brief != "" {
		printBrief(out, ex.Brief)
	}
	watch.WriteLegend(out)
	if ex.Type == content.TypeQuiz {
		fmt.Fprintln(out, dim("Press Enter to start the quiz."))
	}
	fmt.Fprintln(out)
}

// stars renders a difficulty rating such as ★★★☆☆.
func stars(n int) string {
	if n < 0 {
		n = 0
	}
	if n > 5 {
		n = 5
	}
	return yellow(strings.Repeat("★", n)) + dim(strings.Repeat("☆", 5-n))
}

// printBrief renders the exercise task, indented under a yellow marker, so it
// stands out from the surrounding chrome.
func printBrief(out io.Writer, brief string) {
	var buf bytes.Buffer
	if err := renderMarkdown(&buf, brief+"\n"); err != nil {
		buf.Reset()
		buf.WriteString(strings.TrimRight(brief, "\n"))
	}
	lines := strings.Split(buf.String(), "\n")
	// glamour wraps the paragraph in blank lines; drop them so the marker sits
	// on the first line of text.
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	for i, ln := range lines {
		ln = strings.TrimRight(ln, " ")
		if i == 0 {
			fmt.Fprintf(out, "  %s %s\n", yellow("➜"), ln)
		} else {
			fmt.Fprintf(out, "    %s\n", ln)
		}
	}
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

// classifyEsc inspects the bytes that follow a leading ESC. It reports seq=true
// when they form an ANSI escape sequence (arrow and function keys), consuming
// them so they are ignored. Otherwise it returns the byte that followed a bare
// Esc (-1 when none arrived within the window) so the caller can handle it.
func classifyEsc(ch <-chan byte) (follow int, seq bool) {
	nb, ok := recvByte(ch, 50*time.Millisecond)
	if !ok {
		return -1, false
	}
	switch nb {
	case '[':
		for {
			fb, ok := recvByte(ch, 50*time.Millisecond)
			if !ok {
				return -1, true
			}
			if fb >= 0x40 && fb <= 0x7e {
				return -1, true
			}
		}
	case 'O':
		_, _ = recvByte(ch, 50*time.Millisecond)
		return -1, true
	default:
		return int(nb), false
	}
}

func recvByte(ch <-chan byte, d time.Duration) (byte, bool) {
	select {
	case b, ok := <-ch:
		return b, ok
	case <-time.After(d):
		return 0, false
	}
}
