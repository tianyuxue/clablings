// Package engine wraps external commands for containerlab and the container runtime.
package engine

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Result is the outcome of one command execution. A non-zero exit code is
// reported via Code with err == nil; err is non-nil only when the process
// cannot be started or the context is cancelled.
type Result struct {
	Stdout string
	Stderr string
	Code   int
}

// Runner executes external commands; tests substitute a fake.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	res := Result{Stdout: out.String(), Stderr: errb.String()}
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			res.Code = ee.ExitCode()
			return res, nil
		}
		return res, err
	}
	return res, nil
}
