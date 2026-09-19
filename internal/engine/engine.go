package engine

import (
	"context"
	"fmt"
	"time"
)

// Engine wraps containerlab and the container runtime.
type Engine struct {
	runner    Runner
	clabBin   string
	dockerBin string
}

// New builds an Engine using the real command runner.
func New(clabBin, dockerBin string) *Engine {
	return &Engine{runner: execRunner{}, clabBin: clabBin, dockerBin: dockerBin}
}

// NewWithRunner allows injecting a Runner (for tests).
func NewWithRunner(runner Runner, clabBin, dockerBin string) *Engine {
	return &Engine{runner: runner, clabBin: clabBin, dockerBin: dockerBin}
}

// Runner returns the underlying command runner (reused by build, etc.).
func (e *Engine) Runner() Runner { return e.runner }

func (e *Engine) Deploy(ctx context.Context, topoPath, labName string, timeout time.Duration) (Result, error) {
	args := []string{"deploy", "-t", topoPath}
	args = appendLabName(args, labName)
	args = append(args, "--timeout", timeout.String())
	return e.runner.Run(ctx, e.clabBin, args...)
}

func (e *Engine) Destroy(ctx context.Context, topoPath, labName string) (Result, error) {
	args := []string{"destroy", "-t", topoPath}
	args = appendLabName(args, labName)
	args = append(args, "--cleanup")
	return e.runner.Run(ctx, e.clabBin, args...)
}

func (e *Engine) Redeploy(ctx context.Context, topoPath, labName string, timeout time.Duration) (Result, error) {
	args := []string{"redeploy", "-t", topoPath}
	args = appendLabName(args, labName)
	args = append(args, "--cleanup", "--timeout", timeout.String())
	return e.runner.Run(ctx, e.clabBin, args...)
}

// appendLabName adds `--name <labName>` when a lab name is set, so the static
// topology name is overridden and multiple instances cannot collide.
func appendLabName(args []string, labName string) []string {
	if labName == "" {
		return args
	}
	return append(args, "--name", labName)
}

func (e *Engine) Exec(ctx context.Context, container, cmd string) (Result, error) {
	return e.runner.Run(ctx, e.dockerBin, "exec", container, "sh", "-lc", cmd)
}

func (e *Engine) CopyToContainer(ctx context.Context, src, container, dest string) error {
	res, err := e.runner.Run(ctx, e.dockerBin, "cp", src, container+":"+dest)
	if err != nil {
		return err
	}
	if res.Code != 0 {
		return fmt.Errorf("docker cp failed (code=%d): %s", res.Code, res.Stderr)
	}
	return nil
}

// Inspect returns a map from lab name to node list.
func (e *Engine) Inspect(ctx context.Context, topoPath, labName string) (map[string][]InspectNode, error) {
	args := []string{"inspect", "-t", topoPath}
	args = appendLabName(args, labName)
	args = append(args, "-f", "json")
	res, err := e.runner.Run(ctx, e.clabBin, args...)
	if err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("containerlab inspect failed (code=%d): %s", res.Code, res.Stderr)
	}
	return ParseInspect([]byte(res.Stdout))
}

// InspectInterfaces returns a map from container name to interface list.
func (e *Engine) InspectInterfaces(ctx context.Context, labName string) (map[string][]InspectInterface, error) {
	res, err := e.runner.Run(ctx, e.clabBin, "inspect", "interfaces", "--name", labName, "-f", "json")
	if err != nil {
		return nil, err
	}
	if res.Code != 0 {
		return nil, fmt.Errorf("containerlab inspect interfaces failed (code=%d): %s", res.Code, res.Stderr)
	}
	return ParseInspectInterfaces([]byte(res.Stdout))
}
