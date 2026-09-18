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
		t.Fatal("期望 Save 报错")
	}
	matches, err := filepath.Glob(filepath.Join(parent, StateFileName+".tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("不应残留临时文件: %v", matches)
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
		t.Fatalf("version = %d, 期望 1", st.Version)
	}
}
