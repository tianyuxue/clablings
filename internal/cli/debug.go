package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
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
			lib, err := content.LoadWithLocale(resolved, locale(cmd.ErrOrStderr()))
			if err != nil {
				return err
			}
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			ch := chapterAt(lib, wd)
			if ch == nil {
				return fmt.Errorf("not inside a chapter directory; run from the chapter containing chapter.yaml")
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

// chapterAt returns the chapter whose directory matches dir.
func chapterAt(lib *content.Library, dir string) *content.Chapter {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil
	}
	for _, ch := range lib.Chapters {
		chAbs, err := filepath.Abs(ch.Dir)
		if err != nil {
			continue
		}
		if chAbs == abs {
			return ch
		}
	}
	return nil
}
