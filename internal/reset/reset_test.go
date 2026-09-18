package reset

import (
	"context"
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
