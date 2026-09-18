package engine

import (
	"context"
	"testing"
)

func TestExecRunnerCapturesOutputAndExitCode(t *testing.T) {
	r := execRunner{}
	res, err := r.Run(context.Background(), "sh", "-c", "echo out; echo err 1>&2; exit 3")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Stdout != "out\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if res.Stderr != "err\n" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if res.Code != 3 {
		t.Fatalf("code = %d, want 3", res.Code)
	}
}

func TestExecRunnerMissingBinaryReturnsError(t *testing.T) {
	if _, err := (execRunner{}).Run(context.Background(), "definitely-not-a-binary-xyz"); err == nil {
		t.Fatal("expected a start error, got nil")
	}
}
