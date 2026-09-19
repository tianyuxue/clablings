package watch

import "io"

// Key is a recognized watch key press.
type Key int

const (
	KeyEnter Key = iota
	KeyHint
	KeyRedeploy
	KeyFastReset
	KeyQuit
	KeyCleanQuit
)

// ReadKeys reads bytes from r and emits one Key per recognized byte until r
// errors or EOF. Unknown bytes are ignored; a full channel drops the event.
func ReadKeys(r io.Reader, ch chan<- Key) {
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if k, ok := keyFor(buf[0]); ok {
				select {
				case ch <- k:
				default:
				}
			}
		}
		if err != nil {
			return
		}
	}
}

func keyFor(b byte) (Key, bool) {
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
	}
	return 0, false
}
