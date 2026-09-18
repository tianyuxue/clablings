package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
)

// Version 由构建时的 -ldflags 覆盖；默认 dev。
var Version = "dev"

// NewRootCmd 构建 clablings 根命令。
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "clablings",
		Short:         "互动式网络实战与协议通关平台",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newListCmd())
	return cmd
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
		Short: "显示版本信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "clablings %s\n", Version)
			return err
		},
	}
}
