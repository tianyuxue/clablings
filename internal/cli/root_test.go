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

func TestRootHasNoLangFlag(t *testing.T) {
	if NewRootCmd().PersistentFlags().Lookup("lang") != nil {
		t.Fatal("--lang must be removed")
	}
}
