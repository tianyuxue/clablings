package build

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func codeExercise() *content.Exercise {
	ex := &content.Exercise{
		ID: "01", Chapter: "c2", Dir: "/ex",
		ChapterRef: &content.Chapter{TopologyName: "clablings-arp"},
		Type:       content.TypeCodeImplement,
		Build:      &content.BuildSpec{Image: "gcc:13", Command: "make", Output: "arp_responder"},
		Run: &content.RunSpec{
			Node: "host1", Dest: "/opt/clablings/arp_responder",
			Stop:  "pkill -f /opt/clablings/arp_responder || true",
			Start: "nohup /opt/clablings/arp_responder --iface eth1 >/var/log/arp.log 2>&1 &",
		},
	}
	return ex
}

func writeOut(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "arp_responder"), []byte("bin"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDeployCopiesBinary(t *testing.T) {
	fr := &fakeRunner{}
	b := New(fr, "docker", "gcc:13")
	if err := b.Deploy(context.Background(), codeExercise(), "/ex/arp_responder"); err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	want := []string{"docker", "cp", "/ex/arp_responder", "clab-clablings-arp-host1:/opt/clablings/arp_responder"}
	if !reflect.DeepEqual(fr.calls[0], want) {
		t.Fatalf("calls[0] = %v, want %v", fr.calls[0], want)
	}
}

func TestRestartStopsThenStarts(t *testing.T) {
	fr := &fakeRunner{}
	b := New(fr, "docker", "gcc:13")
	if err := b.Restart(context.Background(), codeExercise()); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	wantStop := []string{"docker", "exec", "clab-clablings-arp-host1", "sh", "-lc", "pkill -f /opt/clablings/arp_responder || true"}
	wantStart := []string{"docker", "exec", "clab-clablings-arp-host1", "sh", "-lc", "nohup /opt/clablings/arp_responder --iface eth1 >/var/log/arp.log 2>&1 &"}
	if !reflect.DeepEqual(fr.calls[0], wantStop) {
		t.Fatalf("stop = %v, want %v", fr.calls[0], wantStop)
	}
	if !reflect.DeepEqual(fr.calls[1], wantStart) {
		t.Fatalf("start = %v, want %v", fr.calls[1], wantStart)
	}
}

func TestRestartToleratesStopFailureButNotStartFailure(t *testing.T) {
	fr := &fakeRunner{results: []engine.Result{{Code: 1}, {Code: 1}}}
	b := New(fr, "docker", "gcc:13")
	if err := b.Restart(context.Background(), codeExercise()); err == nil {
		t.Fatal("start failure should return an error")
	}
}

func TestRebuildBuildsDeploysRestarts(t *testing.T) {
	dir := t.TempDir()
	writeOut(t, dir)
	fr := &fakeRunner{}
	b := New(fr, "docker", "gcc:13")
	ex := codeExercise()
	ex.Dir = dir
	if err := b.Rebuild(context.Background(), ex); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
	if len(fr.calls) != 4 {
		t.Fatalf("calls = %d, want 4", len(fr.calls))
	}
	if fr.calls[0][1] != "run" || fr.calls[1][1] != "cp" || fr.calls[2][1] != "exec" {
		t.Fatalf("wrong call order: %v", fr.calls)
	}
}

func TestDeployErrors(t *testing.T) {
	if err := New(&fakeRunner{}, "docker", "gcc:13").Deploy(context.Background(), &content.Exercise{ID: "e1", Chapter: "c1"}, "/bin"); err == nil {
		t.Fatal("expected error for missing run spec")
	}
	fr := &fakeRunner{results: []engine.Result{{Code: 1, Stderr: "cp boom"}}}
	if err := New(fr, "docker", "gcc:13").Deploy(context.Background(), codeExercise(), "/bin"); err == nil {
		t.Fatal("expected deploy failure")
	}
	fr2 := &fakeRunner{errs: []error{errors.New("start failed")}}
	if err := New(fr2, "docker", "gcc:13").Deploy(context.Background(), codeExercise(), "/bin"); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestRestartToleratesStopFailure(t *testing.T) {
	fr := &fakeRunner{results: []engine.Result{{Code: 1}, {Code: 0}}}
	if err := New(fr, "docker", "gcc:13").Restart(context.Background(), codeExercise()); err != nil {
		t.Fatalf("stop failure must be tolerated: %v", err)
	}
	if len(fr.calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(fr.calls))
	}
}

func TestRestartErrors(t *testing.T) {
	if err := New(&fakeRunner{}, "docker", "gcc:13").Restart(context.Background(), &content.Exercise{ID: "e1", Chapter: "c1"}); err == nil {
		t.Fatal("expected error for missing run spec")
	}
	fr := &fakeRunner{errs: []error{nil, errors.New("start failed")}}
	if err := New(fr, "docker", "gcc:13").Restart(context.Background(), codeExercise()); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestRestartWithoutStop(t *testing.T) {
	ex := codeExercise()
	ex.Run.Stop = ""
	fr := &fakeRunner{}
	if err := New(fr, "docker", "gcc:13").Restart(context.Background(), ex); err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if len(fr.calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(fr.calls))
	}
}

func TestRebuildPropagatesBuildFailure(t *testing.T) {
	fr := &fakeRunner{results: []engine.Result{{Code: 1, Stderr: "compile boom"}}}
	ex := codeExercise()
	ex.Dir = t.TempDir()
	if err := New(fr, "docker", "gcc:13").Rebuild(context.Background(), ex); err == nil {
		t.Fatal("expected build failure")
	}
}

func TestContainerNameFallback(t *testing.T) {
	if got := containerName(&content.Exercise{}, "r1"); got != "r1" {
		t.Fatalf("containerName = %q, want r1", got)
	}
	if got := containerName(&content.Exercise{ChapterRef: &content.Chapter{TopologyName: "lab"}}, "r1"); got != "clab-lab-r1" {
		t.Fatalf("containerName = %q", got)
	}
}
