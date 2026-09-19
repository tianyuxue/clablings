package watch

import (
	"golang.org/x/term"
)

// MakeRaw puts fd into raw mode. A nil result means fd is not a TTY, in which
// case keyboard triggers are disabled.
func MakeRaw(fd int) *term.State {
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil
	}
	return st
}
