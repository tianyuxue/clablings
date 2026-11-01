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

func TestResolveLocale(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		env        string
		configured string
		want       string
	}{
		{name: "default is Chinese", want: "zh"},
		{name: "flag wins", flag: "en", env: "zh", configured: "zh", want: "en"},
		{name: "env beats config", env: "en", configured: "zh", want: "en"},
		{name: "config when flag and env are empty", configured: "en", want: "en"},
		{name: "normalizes flag", flag: "zh-CN", want: "zh"},
		{name: "empty config falls back to Chinese", configured: "", want: "zh"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveLocale(tt.flag, tt.env, tt.configured); got != tt.want {
				t.Fatalf("resolveLocale(%q, %q, %q) = %q, want %q", tt.flag, tt.env, tt.configured, got, tt.want)
			}
		})
	}
}
