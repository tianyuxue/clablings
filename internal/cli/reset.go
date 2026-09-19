package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/build"
	"github.com/tianyuxue/clablings/internal/config"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/reset"
	"github.com/tianyuxue/clablings/internal/state"
)

func newResetCmd() *cobra.Command {
	var root string
	var hard bool
	cmd := &cobra.Command{
		Use:   "reset [<exercise_id>]",
		Short: "Reset an exercise (default: current; fast path unless --hard)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			id := ""
			if len(args) == 1 {
				id = args[0]
			}
			return runReset(cmd.Context(), resolved, id, hard)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	cmd.Flags().BoolVar(&hard, "hard", false, "redeploy the chapter topology for a clean slate")
	return cmd
}

func runReset(ctx context.Context, root, id string, hard bool) error {
	lib, err := content.LoadWithLocale(root, locale(os.Stderr))
	if err != nil {
		return err
	}
	ex, err := resolveExercise(lib, id)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: reading config failed, using defaults: %v\n", err)
	}
	eng := engine.New(binOr(cfg.Containerlab.BinPath, "containerlab"), binOr(cfg.Docker.BinPath, "docker"))
	builder := build.New(eng.Runner(), binOr(cfg.Docker.BinPath, "docker"), cfg.Build.DefaultImage)
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng), Rebuilder: builder})
	if hard {
		return res.Hard(ctx, ex)
	}
	return res.Fast(ctx, ex)
}

// resolveExercise resolves an exercise: an empty id uses the current exercise
// from state; otherwise it is resolved by fully qualified or short ID.
func resolveExercise(lib *content.Library, id string) (*content.Exercise, error) {
	if id == "" {
		st, err := state.Load(lib.Root)
		if err != nil {
			return nil, err
		}
		if st.Current == "" {
			return nil, fmt.Errorf("no current exercise; pass an exercise_id or run `clablings run` first")
		}
		id = st.Current
	}
	return lib.Resolve(id)
}
