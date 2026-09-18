package reset

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/tianyuxue/clablings/internal/content"
)

// fakeEngine records calls.
type fakeEngine struct {
	copied    []string
	execs     []string
	redeploys int
}

func (f *fakeEngine) Redeploy(context.Context, *content.Chapter) error {
	f.redeploys++
	return nil
}
func (f *fakeEngine) CopyToContainer(_ context.Context, src, container, dest string) error {
	f.copied = append(f.copied, src+"->"+container+":"+dest)
	return nil
}
func (f *fakeEngine) Exec(_ context.Context, node, cmd string) error {
	f.execs = append(f.execs, node+": "+cmd)
	return nil
}

func TestFastResetConfigFix(t *testing.T) {
	fe := &fakeEngine{}
	r := New(Options{
		Root:   "/repo",
		Engine: fe,
		Git: func(_ context.Context, root string, paths []string) error {
			if root != "/repo" || len(paths) != 1 || paths[0] != "chapters/01/exercises/01/frr1/frr.conf" {
				t.Fatalf("git args: root=%q paths=%v", root, paths)
			}
			return nil
		},
	})
	ch := &content.Chapter{ID: "01", TopologyName: "clablings-ospf-mtu"}
	ex := &content.Exercise{
		ID: "01", Chapter: "01", Type: content.TypeConfigFix,
		Dir:        "/repo/chapters/01/exercises/01",
		Files:      []content.FileMapping{{Source: "frr1/frr.conf", Node: "r1", Dest: "/etc/frr/frr.conf"}},
		Reload:     []content.NodeCmd{{Node: "r1", Cmd: "vtysh -c 'clear ip ospf process'"}},
		ChapterRef: ch,
	}
	if err := r.Fast(context.Background(), ex); err != nil {
		t.Fatalf("Fast: %v", err)
	}
	if len(fe.copied) != 1 || fe.copied[0] != "/repo/chapters/01/exercises/01/frr1/frr.conf->clab-clablings-ospf-mtu-r1:/etc/frr/frr.conf" {
		t.Fatalf("copied = %v", fe.copied)
	}
	if len(fe.execs) != 1 || fe.execs[0] != "clab-clablings-ospf-mtu-r1: vtysh -c 'clear ip ospf process'" {
		t.Fatalf("execs = %v", fe.execs)
	}
}

type fakeRebuilder struct{ called []string }

func (f *fakeRebuilder) Rebuild(_ context.Context, ex *content.Exercise) error {
	f.called = append(f.called, ex.FullID())
	return nil
}

func codeExercise() *content.Exercise {
	ex := &content.Exercise{ID: "01", Chapter: "c1", Type: content.TypeCodeImplement, Dir: "/repo/c1/exercises/01"}
	ex.ChapterRef = &content.Chapter{ID: "c1", TopologyName: "lab"}
	return ex
}

func TestHardResetConfigFix(t *testing.T) {
	fe := &fakeEngine{}
	r := New(Options{Root: "/repo", Engine: fe, Git: func(context.Context, string, []string) error { return nil }})
	ch := &content.Chapter{ID: "01", TopologyName: "clablings-ospf-mtu"}
	ex := &content.Exercise{ID: "01", Chapter: "01", Type: content.TypeConfigFix, Dir: "/repo/chapters/01/exercises/01",
		Files:      []content.FileMapping{{Source: "frr1/frr.conf", Node: "r1", Dest: "/etc/frr/frr.conf"}},
		Reload:     []content.NodeCmd{{Node: "r1", Cmd: "reload"}},
		ChapterRef: ch,
	}
	if err := r.Hard(context.Background(), ex); err != nil {
		t.Fatalf("Hard: %v", err)
	}
	if fe.redeploys != 1 {
		t.Fatalf("redeploys = %d", fe.redeploys)
	}
	if len(fe.copied) != 1 || len(fe.execs) != 1 {
		t.Fatalf("copied=%v execs=%v", fe.copied, fe.execs)
	}
}

func TestHardResetWithoutChapterRefErrors(t *testing.T) {
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return nil }})
	ex := &content.Exercise{ID: "01", Chapter: "c", Type: content.TypeConfigFix}
	if err := r.Hard(context.Background(), ex); err == nil {
		t.Fatal("expected error")
	}
}

func TestFastResetCodeImplementUsesRebuilder(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeRebuilder{}
	r := New(Options{Root: "/repo", Engine: fe, Git: func(context.Context, string, []string) error { return nil }, Rebuilder: fr})
	if err := r.Fast(context.Background(), codeExercise()); err != nil {
		t.Fatalf("Fast: %v", err)
	}
	if len(fr.called) != 1 || fr.called[0] != "c1/01" {
		t.Fatalf("called = %v", fr.called)
	}
	if len(fe.copied) != 0 || len(fe.execs) != 0 {
		t.Fatalf("copy/exec should not run: %v %v", fe.copied, fe.execs)
	}
}

func TestFastResetCodeImplementWithoutRebuilderErrors(t *testing.T) {
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return nil }})
	if err := r.Fast(context.Background(), codeExercise()); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewDefaultsGit(t *testing.T) {
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}})
	if r.opts.Git == nil || reflect.ValueOf(r.opts.Git).Pointer() != reflect.ValueOf(DefaultGitCheckout).Pointer() {
		t.Fatal("Git not defaulted to DefaultGitCheckout")
	}
}

func TestFastResetSkipsGitWhenNoFiles(t *testing.T) {
	called := false
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { called = true; return nil }})
	ex := &content.Exercise{ID: "01", Chapter: "c", Type: content.TypeConfigFix}
	if err := r.Fast(context.Background(), ex); err != nil {
		t.Fatalf("Fast: %v", err)
	}
	if called {
		t.Fatal("Git should not be called with no files")
	}
}

func TestFastResetPropagatesGitError(t *testing.T) {
	sentinel := errors.New("git boom")
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return sentinel }})
	ex := &content.Exercise{ID: "01", Chapter: "c", Type: content.TypeConfigFix, Dir: "/repo/c", Files: []content.FileMapping{{Source: "a"}}}
	if err := r.Fast(context.Background(), ex); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v", err)
	}
}

func TestFastResetSkipsIncompleteSeedMapping(t *testing.T) {
	fe := &fakeEngine{}
	r := New(Options{Root: "/repo", Engine: fe, Git: func(context.Context, string, []string) error { return nil }})
	ex := &content.Exercise{ID: "01", Chapter: "c", Type: content.TypeConfigFix, Dir: "/repo/c", Files: []content.FileMapping{{Source: "a"}}}
	if err := r.Fast(context.Background(), ex); err != nil {
		t.Fatalf("Fast: %v", err)
	}
	if len(fe.copied) != 0 {
		t.Fatalf("copied = %v", fe.copied)
	}
}

func TestDefaultGitCheckoutError(t *testing.T) {
	dir := t.TempDir()
	err := DefaultGitCheckout(context.Background(), dir, []string{"nope.txt"})
	if err == nil || !strings.Contains(err.Error(), "git checkout failed") {
		t.Fatalf("err = %v, want git checkout failed", err)
	}
}

func TestHardResetCodeImplementUsesRebuilder(t *testing.T) {
	fe := &fakeEngine{}
	fr := &fakeRebuilder{}
	r := New(Options{Root: "/repo", Engine: fe, Git: func(context.Context, string, []string) error { return nil }, Rebuilder: fr})
	ex := codeExercise()
	ex.ChapterRef = &content.Chapter{ID: "c1", TopologyName: "lab"}
	if err := r.Hard(context.Background(), ex); err != nil {
		t.Fatalf("Hard: %v", err)
	}
	if fe.redeploys != 1 || len(fr.called) != 1 {
		t.Fatalf("redeploys=%d called=%v", fe.redeploys, fr.called)
	}
}

func TestHardResetCodeImplementWithoutRebuilderErrors(t *testing.T) {
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return nil }})
	ex := codeExercise()
	ex.ChapterRef = &content.Chapter{ID: "c1", TopologyName: "lab"}
	if err := r.Hard(context.Background(), ex); err == nil {
		t.Fatal("expected error")
	}
}

type errRebuilder struct{ err error }

func (e errRebuilder) Rebuild(context.Context, *content.Exercise) error { return e.err }

func TestFastResetPropagatesRebuildError(t *testing.T) {
	sentinel := errors.New("build boom")
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return nil }, Rebuilder: errRebuilder{err: sentinel}})
	if err := r.Fast(context.Background(), codeExercise()); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v", err)
	}
}

func TestHardResetPropagatesRebuildError(t *testing.T) {
	sentinel := errors.New("build boom")
	r := New(Options{Root: "/repo", Engine: &fakeEngine{}, Git: func(context.Context, string, []string) error { return nil }, Rebuilder: errRebuilder{err: sentinel}})
	ex := codeExercise()
	ex.ChapterRef = &content.Chapter{ID: "c1", TopologyName: "lab"}
	if err := r.Hard(context.Background(), ex); !errors.Is(err, sentinel) {
		t.Fatalf("err = %v", err)
	}
}
