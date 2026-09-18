package build

import (
	"context"
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
