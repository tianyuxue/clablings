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

// defaultLanguage is the content language used when no flag, environment
// variable, or config file selects one.
const defaultLanguage = "zh"

// locale resolves content language: --lang > CLABLINGS_LANG > config.language > zh.
func locale(errOut io.Writer) string {
	cfg, err := config.Load()
	if err != nil && errOut != nil {
		fmt.Fprintf(errOut, "warning: reading config failed, using defaults: %v\n", err)
	}
	configured := ""
	if cfg != nil {
		configured = cfg.Language
	}
	return resolveLocale(langFlag, os.Getenv("CLABLINGS_LANG"), configured)
}

// resolveLocale applies the language precedence, falling back to Chinese.
func resolveLocale(flag, env, configured string) string {
	for _, value := range []string{flag, env, configured} {
		if value != "" {
			return content.NormalizeLocale(value)
		}
	}
	return defaultLanguage
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
	cmd.PersistentFlags().StringVar(&langFlag, "lang", "", "content language: en or zh (default: zh)")
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
