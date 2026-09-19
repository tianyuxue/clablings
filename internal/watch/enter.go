package watch

import (
	"golang.org/x/term"
)

// MakeRaw puts fd into raw mode. A nil result means fd is not a TTY and the
// caller should disable the Enter trigger.
func MakeRaw(fd int) *term.State {
	st, err := term.MakeRaw(fd)
	if err != nil {
		return nil
	}
	return st
}
