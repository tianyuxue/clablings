package watch

import (
	"bytes"
	"testing"
	"time"
)

func readOne(t *testing.T, in string) (Key, bool) {
	t.Helper()
	ch := make(chan Key, 4)
	go ReadKeys(bytes.NewReader([]byte(in)), ch)
	select {
	case k := <-ch:
		return k, true
	case <-time.After(time.Second):
		return 0, false
	}
}

func TestReadKeysMappings(t *testing.T) {
	cases := map[byte]Key{
		'\n': KeyEnter, '\r': KeyEnter,
		'h': KeyHint, 'H': KeyHint,
		'r': KeyRedeploy, 'R': KeyRedeploy,
		'f': KeyFastReset, 'F': KeyFastReset,
		'q': KeyQuit, 'Q': KeyQuit, 0x03: KeyQuit,
		'x': KeyCleanQuit, 'X': KeyCleanQuit,
	}
	for b, want := range cases {
		if got, ok := readOne(t, string([]byte{b})); !ok || got != want {
			t.Fatalf("byte %q => (%v,%v), want %v", b, got, ok, want)
		}
	}
}

func TestReadKeysIgnoresUnknown(t *testing.T) {
	ch := make(chan Key, 1)
	go ReadKeys(bytes.NewReader([]byte("z\n")), ch)
	select {
	case k := <-ch:
		if k != KeyEnter {
			t.Fatalf("got %v, want KeyEnter", k)
		}
	case <-time.After(time.Second):
		t.Fatal("expected KeyEnter after ignored byte")
	}
}
