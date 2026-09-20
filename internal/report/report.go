// Package report renders assertion results as English-labeled terminal text or JSON.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/assert"
)

// Text outputs a structured report. Labels are English; assertion names/hints
// come from exercise content and may be in any language.
func Text(w io.Writer, results []assert.Result) {
	for _, r := range results {
		mark := colorize("32", "✓")
		if !r.Passed {
			mark = colorize("31", "✗")
		}
		fmt.Fprintf(w, "%s %s%s\n", mark, label(r.Layer), r.Name)
		if !r.Passed {
			if r.Expected != "" {
				fmt.Fprintf(w, "  Expected: %s\n", r.Expected)
			}
			if r.Actual != "" {
				fmt.Fprintf(w, "  Actual:   %s\n", r.Actual)
			}
			if r.Hint != "" {
				fmt.Fprintf(w, "  Hint:     %s\n", r.Hint)
			}
			if r.Err != nil {
				fmt.Fprintf(w, "  Error:    %v\n", r.Err)
			}
		}
	}
}

// colorize wraps s in an ANSI color when stdout is a terminal.
func colorize(code, s string) string {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func label(layer string) string {
	if layer == "" {
		return ""
	}
	return "[" + layer + "] "
}

type jsonResult struct {
	Exercise  string `json:"exercise"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Layer     string `json:"layer,omitempty"`
	Viewpoint string `json:"viewpoint,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Passed    bool   `json:"passed"`
	Error     string `json:"error,omitempty"`
}

// JSON writes the structured JSON results.
func JSON(w io.Writer, results []assert.Result) error {
	out := make([]jsonResult, 0, len(results))
	for _, r := range results {
		jr := jsonResult{
			Exercise: r.Exercise, Name: r.Name, Type: r.Type, Layer: r.Layer,
			Viewpoint: r.Viewpoint, Expected: r.Expected, Actual: r.Actual,
			Hint: r.Hint, Passed: r.Passed,
		}
		if r.Err != nil {
			jr.Error = r.Err.Error()
		}
		out = append(out, jr)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// AllPassed reports whether all results passed.
func AllPassed(results []assert.Result) bool {
	for _, r := range results {
		if !r.Passed {
			return false
		}
	}
	return true
}
