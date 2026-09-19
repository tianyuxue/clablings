package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// fakeRunner records the call sequence and returns preset results.
type fakeRunner struct {
	calls   [][]string
	results []Result
	errs    []error
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (Result, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	i := len(f.calls) - 1
	if i < len(f.results) {
		var err error
		if i < len(f.errs) {
			err = f.errs[i]
		}
		return f.results[i], err
	}
	return Result{}, nil
}

func TestEngineDeployCommand(t *testing.T) {
	fr := &fakeRunner{}
	e := NewWithRunner(fr, "containerlab", "docker")
	if _, err := e.Deploy(context.Background(), "/tmp/topo.clab.yml", "lab-x", 2*time.Minute); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	want := []string{"containerlab", "deploy", "-t", "/tmp/topo.clab.yml", "--name", "lab-x", "--timeout", "2m0s"}
	if !reflect.DeepEqual(fr.calls[0], want) {
		t.Fatalf("calls[0] = %v, want %v", fr.calls[0], want)
	}
}

func TestEngineDestroyAndRedeployCommands(t *testing.T) {
	fr := &fakeRunner{}
	e := NewWithRunner(fr, "containerlab", "docker")
	_, _ = e.Destroy(context.Background(), "/tmp/topo.clab.yml", "lab-x")
	_, _ = e.Redeploy(context.Background(), "/tmp/topo.clab.yml", "lab-x", time.Minute)

	wantDestroy := []string{"containerlab", "destroy", "-t", "/tmp/topo.clab.yml", "--name", "lab-x", "--cleanup"}
	if !reflect.DeepEqual(fr.calls[0], wantDestroy) {
		t.Fatalf("destroy = %v, want %v", fr.calls[0], wantDestroy)
	}
	wantRedeploy := []string{"containerlab", "redeploy", "-t", "/tmp/topo.clab.yml", "--name", "lab-x", "--cleanup", "--timeout", "1m0s"}
	if !reflect.DeepEqual(fr.calls[1], wantRedeploy) {
		t.Fatalf("redeploy = %v, want %v", fr.calls[1], wantRedeploy)
	}
}

func TestEngineEmptyLabNameOmitsNameFlag(t *testing.T) {
	fr := &fakeRunner{}
	e := NewWithRunner(fr, "containerlab", "docker")
	_, _ = e.Deploy(context.Background(), "/t.yml", "", time.Minute)
	_, _ = e.Destroy(context.Background(), "/t.yml", "")
	_, _ = e.Redeploy(context.Background(), "/t.yml", "", time.Minute)
	_, _ = e.Inspect(context.Background(), "/t.yml", "")
	wants := [][]string{
		{"containerlab", "deploy", "-t", "/t.yml", "--timeout", "1m0s"},
		{"containerlab", "destroy", "-t", "/t.yml", "--cleanup"},
		{"containerlab", "redeploy", "-t", "/t.yml", "--cleanup", "--timeout", "1m0s"},
		{"containerlab", "inspect", "-t", "/t.yml", "-f", "json"},
	}
	if len(fr.calls) != len(wants) {
		t.Fatalf("calls = %v", fr.calls)
	}
	for i := range wants {
		if !reflect.DeepEqual(fr.calls[i], wants[i]) {
			t.Fatalf("calls[%d] = %v, want %v", i, fr.calls[i], wants[i])
		}
	}
}

func TestEngineExecAndCopyCommands(t *testing.T) {
	fr := &fakeRunner{}
	e := NewWithRunner(fr, "containerlab", "docker")
	if _, err := e.Exec(context.Background(), "clab-x-r1", "vtysh -c 'show version'"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if err := e.CopyToContainer(context.Background(), "/tmp/a.conf", "clab-x-r1", "/etc/frr/frr.conf"); err != nil {
		t.Fatalf("Copy: %v", err)
	}
	wantExec := []string{"docker", "exec", "clab-x-r1", "sh", "-lc", "vtysh -c 'show version'"}
	if !reflect.DeepEqual(fr.calls[0], wantExec) {
		t.Fatalf("exec = %v, want %v", fr.calls[0], wantExec)
	}
	wantCopy := []string{"docker", "cp", "/tmp/a.conf", "clab-x-r1:/etc/frr/frr.conf"}
	if !reflect.DeepEqual(fr.calls[1], wantCopy) {
		t.Fatalf("copy = %v, want %v", fr.calls[1], wantCopy)
	}
}

func TestEngineCommandsFailOnNonZeroCode(t *testing.T) {
	fr := &fakeRunner{results: []Result{
		{Code: 1, Stderr: "inspect boom"},
		{Code: 1, Stderr: "ifaces boom"},
		{Code: 1, Stderr: "cp boom"},
	}}
	e := NewWithRunner(fr, "containerlab", "docker")

	if _, err := e.Inspect(context.Background(), "/t.yml", ""); err == nil || !strings.Contains(err.Error(), "inspect boom") {
		t.Fatalf("Inspect err = %v, want contains %q", err, "inspect boom")
	}
	if _, err := e.InspectInterfaces(context.Background(), "lab"); err == nil || !strings.Contains(err.Error(), "ifaces boom") {
		t.Fatalf("InspectInterfaces err = %v, want contains %q", err, "ifaces boom")
	}
	if err := e.CopyToContainer(context.Background(), "/a", "c", "/b"); err == nil || !strings.Contains(err.Error(), "cp boom") {
		t.Fatalf("CopyToContainer err = %v, want contains %q", err, "cp boom")
	}
}

func TestEnginePropagatesRunnerError(t *testing.T) {
	sentinel := errors.New("start failed")
	fr := &fakeRunner{results: []Result{{}}, errs: []error{sentinel}}
	e := NewWithRunner(fr, "containerlab", "docker")
	if _, err := e.Inspect(context.Background(), "/t.yml", ""); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}
