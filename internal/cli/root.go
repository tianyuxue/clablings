package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/config"
	"github.com/tianyuxue/clablings/internal/content"
)

// exitCodeError carries a process exit code from a command to Execute.
type exitCodeError struct{ code int }

func (e *exitCodeError) Error() string { return fmt.Sprintf("exit code %d", e.code) }

// Version is overridden at build time via -ldflags; defaults to dev.
var Version = "dev"

var langFlag string

// locale resolves content language: --lang > CLABLINGS_LANG > config.language > en.
func locale(errOut io.Writer) string {
	if langFlag != "" {
		return content.NormalizeLocale(langFlag)
	}
	if v := os.Getenv("CLABLINGS_LANG"); v != "" {
		return content.NormalizeLocale(v)
	}
	// Config is read again here (commands may also load it); this keeps the
	// lookup self-contained at the cost of one extra read.
	cfg, err := config.Load()
	if err != nil && errOut != nil {
		fmt.Fprintf(errOut, "warning: reading config failed, using defaults: %v\n", err)
	}
	if cfg != nil {
		return content.NormalizeLocale(cfg.Language)
	}
	return content.DefaultLocale
}

// NewRootCmd builds the clablings root command.
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "clablings",
		Short:         "Interactive network practice and protocol mastery platform",
		SilenceErrors: true,
		SilenceUsage:  true,
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
	cmd.PersistentFlags().StringVar(&langFlag, "lang", "", "content language: en or zh (default: en)")
	return cmd
}

// Execute runs the root command and returns the process exit code to main.
func Execute() int {
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
