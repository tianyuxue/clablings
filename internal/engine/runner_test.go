package engine

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
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
	if _, err := (execRunner{}).Run(context.Background(), "definitely-not-a-binary-xyz"); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("err = %v, want exec.ErrNotFound", err)
	}
}

func TestExecRunnerContextCanceledReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := (execRunner{}).Run(ctx, "sh", "-c", "exec sleep 5")
		errc <- err
	}()
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
