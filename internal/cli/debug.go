package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/state"
)

func newDebugCmd() *cobra.Command {
	var root string
	var useSudo bool
	cmd := &cobra.Command{
		Use:          "debug",
		Short:        "Launch clab-tui on the current chapter's shared topology",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			lib, err := content.Load(resolved)
			if err != nil {
				return err
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			ch := chapterForInvocation(lib, resolved, wd)
			if ch == nil {
				return fmt.Errorf("not inside a chapter directory and no current exercise; run from a chapter or start clablings watch first")
			}
			if !ch.HasTopology() {
				return fmt.Errorf("chapter %s has no topology; clab-tui requires one", ch.ID)
			}
			bin, err := exec.LookPath("clab-tui")
			if err != nil {
				return fmt.Errorf("clab-tui not found in PATH: %w", err)
			}
			runner := bin
			if useSudo {
				sudoBin, err := exec.LookPath("sudo")
				if err != nil {
					return fmt.Errorf("sudo not found in PATH: %w", err)
				}
				runner = sudoBin
			}
			topo, err := engine.PrepareTopology(ch)
			if err != nil {
				return fmt.Errorf("chapter %s: %w", ch.ID, err)
			}
			launchArgs := []string{"--file", topo}
			if useSudo {
				// sudo must receive the resolved absolute path; root's PATH may differ.
				launchArgs = append([]string{bin}, launchArgs...)
			}
			c := exec.CommandContext(cmd.Context(), runner, launchArgs...)
			c.Stdin = cmd.InOrStdin()
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			return c.Run()
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	cmd.Flags().BoolVar(&useSudo, "sudo", false, "launch clab-tui via sudo (root)")
	return cmd
}

// chapterForInvocation resolves the chapter for commands launched from the
// current directory: the enclosing chapter first (so exercise subdirectories
// work), then the current exercise recorded by watch.
func chapterForInvocation(lib *content.Library, root, dir string) *content.Chapter {
	if ch := chapterAt(lib, dir); ch != nil {
		return ch
	}
	st, err := state.Load(root)
	if err != nil || st == nil || st.Current == "" {
		return nil
	}
	if ex := lib.ByID[st.Current]; ex != nil {
		return ex.ChapterRef
	}
	return nil
}

// chapterAt returns the chapter whose directory contains dir, walking up from
// dir. This lets commands run from an exercise subdirectory.
func chapterAt(lib *content.Library, dir string) *content.Chapter {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	for {
		if ch := chapterDir(lib, abs); ch != nil {
			return ch
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil
		}
		abs = parent
	}
}

// chapterDir returns the chapter whose directory is exactly dir.
func chapterDir(lib *content.Library, dir string) *content.Chapter {
	for _, ch := range lib.Chapters {
		chAbs, err := filepath.Abs(ch.Dir)
		if err != nil {
			continue
		}
		if chAbs == dir {
			return ch
		}
	}
	return nil
}
