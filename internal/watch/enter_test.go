package watch

import (
	"bytes"
	"testing"
	"time"
)

func TestReadEnterSignalsOnNewline(t *testing.T) {
	ch := make(chan struct{}, 1)
	r := bytes.NewReader([]byte("x\ny"))
	go ReadEnter(r, ch)

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected a signal for LF")
	}
	select {
	case <-ch:
		t.Fatal("must not signal twice for one newline")
	default:
	}
}

func TestReadEnterSignalsOnCarriageReturn(t *testing.T) {
	ch := make(chan struct{}, 1)
	go ReadEnter(bytes.NewReader([]byte("x\ry")), ch)
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("expected a signal for CR")
	}
}

func TestReadEnterSignalsForEachNewline(t *testing.T) {
	ch := make(chan struct{}, 2)
	go ReadEnter(bytes.NewReader([]byte("a\nb\n")), ch)
	for i := 0; i < 2; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("expected signal %d", i+1)
		}
	}
}
