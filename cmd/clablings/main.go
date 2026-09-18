package main

import (
	"fmt"
	"os"

	"github.com/tianyuxue/clablings/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "clablings: 错误: %v\n", err)
		os.Exit(1)
	}
}
