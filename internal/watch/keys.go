package watch

import (
	"errors"
	"io"
)

// ErrQuizAborted is returned when the user aborts an interactive quiz.
var ErrQuizAborted = errors.New("quiz aborted")

// ErrQuizUnsupported is returned when a quiz is reached without an interactive
// handler (e.g. non-TTY runs).
var ErrQuizUnsupported = errors.New("interactive quiz not supported")

// Key is a recognized watch key press.
type Key int

const (
	KeyEnter Key = iota
	KeyHint
	KeyRedeploy
	KeyFastReset
	KeyQuit
	KeyCleanQuit
	KeyConcept
)

// KeyFor maps a byte to a Key; ok is false for unmapped bytes.
func KeyFor(b byte) (Key, bool) {
	switch b {
	case '\n', '\r':
		return KeyEnter, true
	case 'h', 'H':
		return KeyHint, true
	case 'r', 'R':
		return KeyRedeploy, true
	case 'f', 'F':
		return KeyFastReset, true
	case 'q', 'Q', 0x03:
		return KeyQuit, true
	case 'x', 'X':
		return KeyCleanQuit, true
	case 'c', 'C':
		return KeyConcept, true
	}
	return 0, false
}

// ReadBytes copies single bytes from r into ch until r errors or EOF. A full
// channel drops the byte. This is the single reader that owns stdin; callers
// demultiplex bytes into keys or quiz input.
func ReadBytes(r io.Reader, ch chan<- byte) {
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			select {
			case ch <- buf[0]:
			default:
			}
		}
		if err != nil {
			return
		}
	}
}
