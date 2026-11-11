package state

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingReturnsFresh(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Version != 1 || st.Exercises == nil {
		t.Fatalf("state = %+v", st)
	}
}

func TestInvokerIDs(t *testing.T) {
	t.Setenv("SUDO_UID", "1000")
	t.Setenv("SUDO_GID", "1001")
	uid, gid, ok := invokerIDs()
	if !ok || uid != 1000 || gid != 1001 {
		t.Fatalf("invokerIDs = (%d, %d, %v), want (1000, 1001, true)", uid, gid, ok)
	}

	t.Setenv("SUDO_UID", "not-a-number")
	if _, _, ok := invokerIDs(); ok {
		t.Fatal("invokerIDs accepted a non-numeric SUDO_UID")
	}

	t.Setenv("SUDO_UID", "")
	t.Setenv("SUDO_GID", "")
	if _, _, ok := invokerIDs(); ok {
		t.Fatal("invokerIDs accepted missing env vars")
	}
}

func TestCommitStateChownsTempBeforeRename(t *testing.T) {
	oldEuid, oldChown, oldRename := geteuid, chownFile, renameFile
	defer func() { geteuid, chownFile, renameFile = oldEuid, oldChown, oldRename }()
	geteuid = func() int { return 0 }
	t.Setenv("SUDO_UID", "123")
	t.Setenv("SUDO_GID", "456")

	var order []string
	var gotUID, gotGID int
	chownFile = func(path string, uid, gid int) error {
		order = append(order, "chown "+path)
		gotUID, gotGID = uid, gid
		return nil
	}
	renameFile = func(oldPath, newPath string) error {
		order = append(order, "rename "+oldPath+" -> "+newPath)
		return nil
	}

	if err := commitState("/lib/.state.tmp", "/lib/.state"); err != nil {
		t.Fatalf("commitState: %v", err)
	}
	want := []string{"chown /lib/.state.tmp", "rename /lib/.state.tmp -> /lib/.state"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	if gotUID != 123 || gotGID != 456 {
		t.Fatalf("chown ids = (%d, %d), want (123, 456)", gotUID, gotGID)
	}
}

func TestCommitStateSkipsChownOutsideSudo(t *testing.T) {
	oldEuid, oldChown, oldRename := geteuid, chownFile, renameFile
	defer func() { geteuid, chownFile, renameFile = oldEuid, oldChown, oldRename }()
	geteuid = func() int { return 1000 }
	t.Setenv("SUDO_UID", "123")
	t.Setenv("SUDO_GID", "456")

	chowned := false
	chownFile = func(string, int, int) error { chowned = true; return nil }
	renamed := false
	renameFile = func(string, string) error { renamed = true; return nil }

	if err := commitState("tmp", "final"); err != nil {
		t.Fatalf("commitState: %v", err)
	}
	if chowned {
		t.Fatal("non-root run must not chown")
	}
	if !renamed {
		t.Fatal("commitState did not rename")
	}
}

func TestCommitStateDoesNotRenameWhenChownFails(t *testing.T) {
	oldEuid, oldChown, oldRename := geteuid, chownFile, renameFile
	defer func() { geteuid, chownFile, renameFile = oldEuid, oldChown, oldRename }()
	geteuid = func() int { return 0 }
	t.Setenv("SUDO_UID", "123")
	t.Setenv("SUDO_GID", "456")

	chownFile = func(string, int, int) error { return errors.New("chown failed") }
	renamed := false
	renameFile = func(string, string) error { renamed = true; return nil }

	if err := commitState("tmp", "final"); err == nil {
		t.Fatal("commitState accepted a failed chown")
	}
	if renamed {
		t.Fatal("commitState renamed a file whose owner was not fixed")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, _ := Load(dir)
	st.SetCurrent("01-ospf-mtu/01_mtu")
	st.Upsert("01-ospf-mtu/01_mtu", StatusPassed, 3)
	if err := st.Save(dir); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Current != "01-ospf-mtu/01_mtu" {
		t.Fatalf("current = %q", got.Current)
	}
	es := got.Exercises["01-ospf-mtu/01_mtu"]
	if es.Status != StatusPassed || es.Attempts != 3 {
		t.Fatalf("exercise state = %+v", es)
	}
}

func TestLoadCorruptQuarantines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StateFileName)
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := Load(dir)
	var corrupt *CorruptError
	if !errors.As(err, &corrupt) {
		t.Fatalf("err = %v, want CorruptError", err)
	}
	if st.Version != 1 {
		t.Fatalf("want usable empty state, got %+v", st)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("want quarantined file to exist: %v", err)
	}
}

func TestGetHitAndMiss(t *testing.T) {
	st := newState()
	if got := st.Get("missing"); got.Status != StatusTodo {
		t.Fatalf("miss status = %q", got.Status)
	}
	st.Upsert("x", StatusInProgress, 2)
	got := st.Get("x")
	if got.Status != StatusInProgress || got.Attempts != 2 {
		t.Fatalf("hit = %+v", got)
	}

	var nilMap State
	if got := nilMap.Get("x"); got.Status != StatusTodo {
		t.Fatalf("nil map miss status = %q", got.Status)
	}
}

func TestSaveMissingRootErrorsNoTempLitter(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "does-not-exist")
	st := newState()
	if err := st.Save(root); err == nil {
		t.Fatal("want Save to error")
	}
	matches, err := filepath.Glob(filepath.Join(parent, StateFileName+".tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temp files should not remain: %v", matches)
	}
}

func TestLoadNormalizesVersionZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StateFileName)
	data := []byte(`{"version":0,"exercises":{"a":{"status":"passed","attempts":1}}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Version != 1 {
		t.Fatalf("version = %d, want 1", st.Version)
	}
}
