package watch

import (
	"bytes"
	"fmt"
	"io"

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

// WriteLegend prints the watch hot-key legend.
func WriteLegend(w io.Writer) {
	fmt.Fprintln(w, "[Enter] verify")
	fmt.Fprintln(w, "[h] hint   [c] concept")
	fmt.Fprintln(w, "[p] prev   [n] next")
	fmt.Fprintln(w, "[r] redeploy   [f] fast reset")
	fmt.Fprintln(w, "[q] quit   [x] clean quit   [Ctrl+C] quit")
}
