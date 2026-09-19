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
	cmd := &cobra.Command{
		Use:   "debug",
		Short: "Launch clab-tui on the current chapter's shared topology",
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
			ch := chapterAt(lib, wd)
			if ch == nil {
				return fmt.Errorf("not inside a chapter directory; run from the chapter containing chapter.yaml")
			}
			topo, err := engine.PrepareTopology(ch)
			if err != nil {
				return fmt.Errorf("chapter %s: %w", ch.ID, err)
			}
			bin, err := exec.LookPath("clab-tui")
			if err != nil {
				return fmt.Errorf("clab-tui not found in PATH: %w", err)
			}
			c := exec.CommandContext(cmd.Context(), bin, "--file", topo)
			c.Stdin = os.Stdin
			c.Stdout = cmd.OutOrStdout()
			c.Stderr = cmd.ErrOrStderr()
			return c.Run()
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
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
