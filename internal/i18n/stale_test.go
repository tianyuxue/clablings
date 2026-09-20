package i18n

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStaleSourceNewer(t *testing.T) {
	times := map[string]time.Time{
		"en": time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
		"zh": time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	last := func(p string) (time.Time, bool, error) { return times[p], true, nil }
	stale, ok, err := Stale(last, "en", "zh")
	if err != nil || !ok || !stale {
		t.Fatalf("stale=%v ok=%v err=%v, want stale", stale, ok, err)
	}
}

func TestStaleTranslationNewer(t *testing.T) {
	times := map[string]time.Time{
		"en": time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		"zh": time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}
	last := func(p string) (time.Time, bool, error) { return times[p], true, nil }
	stale, ok, err := Stale(last, "en", "zh")
	if err != nil || !ok || stale {
		t.Fatalf("stale=%v ok=%v err=%v, want not stale but comparable", stale, ok, err)
	}
}

func TestStaleUnknownCommit(t *testing.T) {
	last := func(p string) (time.Time, bool, error) { return time.Time{}, false, nil }
	stale, ok, err := Stale(last, "en", "zh")
	if err != nil || ok || stale {
		t.Fatalf("stale=%v ok=%v err=%v, want skip", stale, ok, err)
	}
}

func TestStalePropagatesError(t *testing.T) {
	boom := errors.New("boom")
	last := func(p string) (time.Time, bool, error) { return time.Time{}, false, boom }
	if _, _, err := Stale(last, "en", "zh"); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}

func TestGitLastCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	git := func(env []string, args ...string) {
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	git(nil, "init")
	git(nil, "config", "user.email", "t@example.com")
	git(nil, "config", "user.name", "t")
	writeFile(t, filepath.Join(dir, "a.md"), "a")
	git([]string{"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2020-01-01T00:00:00Z"}, "add", "a.md")
	git([]string{"GIT_AUTHOR_DATE=2020-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2020-01-01T00:00:00Z"}, "commit", "-m", "a")
	writeFile(t, filepath.Join(dir, "b.md"), "b")
	git([]string{"GIT_AUTHOR_DATE=2021-06-01T00:00:00Z", "GIT_COMMITTER_DATE=2021-06-01T00:00:00Z"}, "add", "b.md")
	git([]string{"GIT_AUTHOR_DATE=2021-06-01T00:00:00Z", "GIT_COMMITTER_DATE=2021-06-01T00:00:00Z"}, "commit", "-m", "b")

	last := GitLastCommit(dir)
	ta, ok, err := last("a.md")
	if err != nil || !ok {
		t.Fatalf("a.md: ok=%v err=%v", ok, err)
	}
	tb, ok, err := last("b.md")
	if err != nil || !ok {
		t.Fatalf("b.md: ok=%v err=%v", ok, err)
	}
	if !tb.After(ta) {
		t.Fatalf("b=%v should be newer than a=%v", tb, ta)
	}

	stale, ok, err := Stale(last, "b.md", "a.md")
	if err != nil || !ok || !stale {
		t.Fatalf("stale=%v ok=%v err=%v, want b.md newer than a.md", stale, ok, err)
	}
}

func writeFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
