package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	cmd := NewRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "clablings ") {
		t.Fatalf("output = %q, want to contain %q", out.String(), "clablings ")
	}
}

func TestRootHasLangFlag(t *testing.T) {
	root := NewRootCmd()
	if root.PersistentFlags().Lookup("lang") == nil {
		t.Fatal("root command is missing the --lang persistent flag")
	}
}
