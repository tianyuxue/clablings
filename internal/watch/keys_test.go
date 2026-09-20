package watch

import (
	"bytes"
	"testing"
	"time"
)

func TestKeyForMappings(t *testing.T) {
	cases := map[byte]Key{
		'\n': KeyEnter, '\r': KeyEnter,
		'h': KeyHint, 'H': KeyHint,
		'r': KeyRedeploy, 'R': KeyRedeploy,
		'f': KeyFastReset, 'F': KeyFastReset,
		'q': KeyQuit, 'Q': KeyQuit, 0x03: KeyQuit,
		'x': KeyCleanQuit, 'X': KeyCleanQuit,
		'c': KeyConcept, 'C': KeyConcept,
		'n': KeyNext, 'N': KeyNext,
		'p': KeyPrev, 'P': KeyPrev,
		0x1b: KeyCancel,
	}
	for b, want := range cases {
		if got, ok := KeyFor(b); !ok || got != want {
			t.Fatalf("KeyFor(%q) = (%v,%v), want %v", b, got, ok, want)
		}
	}
	if _, ok := KeyFor('z'); ok {
		t.Fatal("unknown byte must not map")
	}
}

func TestReadBytes(t *testing.T) {
	ch := make(chan byte, 4)
	done := make(chan struct{})
	go func() { ReadBytes(bytes.NewReader([]byte("ab")), ch); close(done) }()
	select {
	case b := <-ch:
		if b != 'a' {
			t.Fatalf("got %q", b)
		}
	case <-time.After(time.Second):
		t.Fatal("no byte")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ReadBytes did not return on EOF")
	}
}
