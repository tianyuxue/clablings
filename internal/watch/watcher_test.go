package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
)

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherDetectsChangeWithoutSentinel(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.conf"), "ok\n")
	ex := &content.Exercise{Dir: dir, Files: []content.FileMapping{{Source: "a.conf"}}}

	w := NewWatcher(ex, 20*time.Millisecond)
	if _, err := w.Scan(); err != nil {
		t.Fatalf("first Scan: %v", err)
	}

	time.Sleep(30 * time.Millisecond)
	write(t, filepath.Join(dir, "a.conf"), "changed\n")
	changed, err := w.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !changed {
		t.Fatal("expected a detected change")
	}
}

func TestWatcherSentinelGatesTrigger(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.c"), "// I AM NOT DONE\n")
	ex := &content.Exercise{Dir: dir, Files: []content.FileMapping{{Source: "a.c"}}}

	w := NewWatcher(ex, 10*time.Millisecond)
	_, _ = w.Scan()
	time.Sleep(15 * time.Millisecond)
	write(t, filepath.Join(dir, "a.c"), "// I AM NOT DONE\nint main(){}\n")
	changed, _ := w.Scan()
	if !changed {
		t.Fatal("expected a detected change")
	}
	if w.ReadyToVerify() {
		t.Fatal("sentinel still present; must not be ready")
	}

	time.Sleep(15 * time.Millisecond)
	write(t, filepath.Join(dir, "a.c"), "int main(){}\n")
	_, _ = w.Scan()
	deadline := time.Now().Add(2 * time.Second)
	for !w.ReadyToVerify() {
		if time.Now().After(deadline) {
			t.Fatal("sentinel removed and debounce elapsed, but still not ready")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestWatcherScansAllFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.c"), "// I AM NOT DONE\n")
	write(t, filepath.Join(dir, "b.h"), "x\n")
	ex := &content.Exercise{Dir: dir, Files: []content.FileMapping{{Source: "a.c"}, {Source: "b.h"}}}

	w := NewWatcher(ex, 10*time.Millisecond)
	if _, err := w.Scan(); err != nil {
		t.Fatalf("first Scan: %v", err)
	}
	if w.ReadyToVerify() {
		t.Fatal("a.c still has sentinel; must not be ready")
	}
	time.Sleep(15 * time.Millisecond)
	write(t, filepath.Join(dir, "b.h"), "y\n")
	changed, err := w.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !changed {
		t.Fatal("later file change must be detected even while an earlier file keeps the sentinel")
	}
	if w.ReadyToVerify() {
		t.Fatal("a.c still has sentinel; must not be ready")
	}
}

func TestWatcherMissingFileErrorsWithoutLosingPriorChange(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "a.c"), "one\n")
	write(t, filepath.Join(dir, "b.c"), "two\n")
	ex := &content.Exercise{Dir: dir, Files: []content.FileMapping{{Source: "a.c"}, {Source: "b.c"}}}
	w := NewWatcher(ex, 10*time.Millisecond)
	if _, err := w.Scan(); err != nil {
		t.Fatalf("first Scan: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	write(t, filepath.Join(dir, "a.c"), "one-changed\n")
	if err := os.Remove(filepath.Join(dir, "b.c")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Scan(); err == nil {
		t.Fatal("expected error for missing watched file")
	}
	write(t, filepath.Join(dir, "b.c"), "two\n")
	time.Sleep(15 * time.Millisecond)
	changed, err := w.Scan()
	if err != nil {
		t.Fatalf("Scan after restore: %v", err)
	}
	if !changed {
		t.Fatal("change to a.c must be re-detected after a failed scan")
	}
}
