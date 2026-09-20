package watch

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/content"
)

// CRLFWriter turns every bare "\n" into "\r\n". It is used while stdin is in
// raw mode, where the terminal no longer performs this translation.
type CRLFWriter struct {
	w      io.Writer
	lastCR bool
}

// NewCRLFWriter wraps w.
func NewCRLFWriter(w io.Writer) *CRLFWriter { return &CRLFWriter{w: w} }

func (c *CRLFWriter) Write(p []byte) (int, error) {
	var buf bytes.Buffer
	for _, b := range p {
		if b == '\n' && !c.lastCR {
			buf.WriteByte('\r')
		}
		buf.WriteByte(b)
		c.lastCR = b == '\r'
	}
	if _, err := c.w.Write(buf.Bytes()); err != nil {
		return 0, err
	}
	return len(p), nil
}

// WriteHints prints the exercise-level hints.
func WriteHints(w io.Writer, ex *content.Exercise) {
	if ex == nil || len(ex.Hints) == 0 {
		fmt.Fprintln(w, "no hints for this exercise")
		return
	}
	fmt.Fprintln(w, "hints:")
	for i, h := range ex.Hints {
		fmt.Fprintf(w, "  %d. %s\n", i+1, h)
	}
}

// WriteLegend prints the watch hot-key legend, with key names highlighted when
// stdout is a terminal.
func WriteLegend(w io.Writer) {
	k := func(s string) string {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			return "\x1b[1;36m" + s + "\x1b[0m"
		}
		return s
	}
	fmt.Fprintln(w, k("[Enter]")+" verify")
	fmt.Fprintln(w, k("[h]")+" hint   "+k("[c]")+" concept")
	fmt.Fprintln(w, k("[p]")+" prev   "+k("[n]")+" next")
	fmt.Fprintln(w, k("[r]")+" redeploy   "+k("[f]")+" fast reset")
	fmt.Fprintln(w, k("[q]")+" quit   "+k("[x]")+" clean quit   "+k("[Ctrl+C]")+" quit")
}
