package cli

import (
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:   "run <exercise_id>",
		Short: "Jump to an exercise and enter the watch loop",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			return runWatch(cmd.Context(), resolved, args[0], cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	return cmd
}
