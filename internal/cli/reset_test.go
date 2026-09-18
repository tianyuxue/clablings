package cli

import (
	"strings"
	"testing"
)

func TestResetCommand(t *testing.T) {
	cmd := newResetCmd()
	if !strings.HasPrefix(cmd.Use, "reset") {
		t.Fatalf("Use = %q", cmd.Use)
	}
	if cmd.Flags().Lookup("hard") == nil {
		t.Fatal("missing --hard flag")
	}
}

func TestResetCmdRegistered(t *testing.T) {
	for _, c := range NewRootCmd().Commands() {
		if c.Name() == "reset" {
			return
		}
	}
	t.Fatal("reset command not registered")
}
