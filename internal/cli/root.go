package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
)

// exitCodeError carries a process exit code from a command to Execute.
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exit code %d", e.code) }

// Version is overridden at build time via -ldflags; defaults to dev.
var Version = "dev"

var sudoFlag bool

// NewRootCmd builds the clablings root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "clablings",
		Short:         "Interactive network practice and protocol mastery platform",
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if !sudoFlag || os.Geteuid() == 0 {
				return nil
			}
			return elevate(os.Args[1:])
		},
	}
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newWatchCmd())
	cmd.AddCommand(newRunCmd())
	cmd.AddCommand(newResetCmd())
	cmd.AddCommand(newTestCmd())
	cmd.AddCommand(newDebugCmd())
	cmd.AddCommand(newReadCmd())
	cmd.AddCommand(newLintCmd())
	cmd.PersistentFlags().BoolVar(&sudoFlag, "sudo", false, "re-run this command as root via sudo")
	return cmd
}

// Execute runs the root command and returns the process exit code to main.
func Execute() int {
	// A --sudo before the subcommand elevates clablings itself. A --sudo after
	// `debug` keeps its local meaning (launch clab-tui via sudo), so it must not
	// be pre-parsed here.
	if shouldElevate(os.Args[1:], os.Geteuid()) {
		if err := elevate(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "clablings: error: %v\n", err)
			return 1
		}
	}
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		var ec *exitCodeError
		if errors.As(err, &ec) {
			return ec.code
		}
		fmt.Fprintf(os.Stderr, "clablings: error: %v\n", err)
		return 1
	}
	return 0
}

func resolveRoot(flag string) (string, error) {
	if flag != "" {
		return content.FindRoot(flag)
	}
	if env := os.Getenv("CLABLINGS_ROOT"); env != "" {
		return content.FindRoot(env)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return content.FindRoot(wd)
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "clablings %s\n", Version)
			return err
		},
	}
}
