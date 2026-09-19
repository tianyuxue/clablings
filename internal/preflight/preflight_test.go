package preflight

import (
	"context"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/engine"
)

type fakeRunner struct {
	results map[string]engine.Result
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (engine.Result, error) {
	key := name + " " + strings.Join(args, " ")
	if r, ok := f.results[key]; ok {
		return r, nil
	}
	return engine.Result{Code: 1}, nil
}

func TestCheckReportsMissingDocker(t *testing.T) {
	fr := &fakeRunner{results: map[string]engine.Result{}}
	err := Check(context.Background(), fr, "containerlab", "docker", nil)
	if err == nil || !strings.Contains(err.Error(), "docker") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckAllPasses(t *testing.T) {
	fr := &fakeRunner{results: map[string]engine.Result{
		"docker info":                 {Code: 0},
		"containerlab version":        {Code: 0},
		"docker image inspect gcc:13": {Code: 0},
	}}
	if err := Check(context.Background(), fr, "containerlab", "docker", []string{"gcc:13"}); err != nil {
		t.Fatalf("Check: %v", err)
	}
}

func TestCheckReportsMissingImage(t *testing.T) {
	fr := &fakeRunner{results: map[string]engine.Result{
		"docker info":          {Code: 0},
		"containerlab version": {Code: 0},
	}}
	err := Check(context.Background(), fr, "containerlab", "docker", []string{"gcc:13"})
	if err == nil || !strings.Contains(err.Error(), "gcc:13") {
		t.Fatalf("err = %v", err)
	}
}
