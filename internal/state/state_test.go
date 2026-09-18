package state

import (
	"errors"
	"os"
	"path/filepath"
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
		t.Fatalf("err = %v, 期望 CorruptError", err)
	}
	if st.Version != 1 {
		t.Fatalf("期望返回可用空状态，实际 %+v", st)
	}
	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("期望隔离文件存在: %v", err)
	}
}
