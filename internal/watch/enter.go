package watch

import (
	"io"

	"golang.org/x/term"
)

// ReadEnter reads bytes from r and signals ch once per newline (LF or CR),
// until r returns an error or EOF. It is intended for a raw-mode stdin; the
// caller is responsible for putting stdin into raw mode.
func ReadEnter(r io.Reader, ch chan<- struct{}) {
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 && (buf[0] == '\n' || buf[0] == '\r') {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		if err != nil {
			return
		}
	}
}

// MakeRaw puts fd into raw mode. A nil result means fd is not a TTY and the
// caller should disable the Enter trigger.
func MakeRaw(fd int) *term.State {
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil
	}
	return st
}
