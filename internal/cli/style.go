package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func colorOn() bool { return term.IsTerminal(int(os.Stdout.Fd())) }

func ansi(code, s string) string {
	if !colorOn() {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func bold(s string) string   { return ansi("1", s) }
func dim(s string) string    { return ansi("2", s) }
func green(s string) string  { return ansi("32", s) }
func red(s string) string    { return ansi("31", s) }
func yellow(s string) string { return ansi("33", s) }
func cyan(s string) string   { return ansi("36", s) }

// progressBar renders a rustlings-style bar like "############-------- 4/7".
func progressBar(i, n int) string {
	const width = 20
	if n <= 0 {
		n = 1
	}
	if i < 0 {
		i = 0
	}
	if i > n {
		i = n
	}
	filled := i * width / n
	bar := strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	if colorOn() {
		bar = green(strings.Repeat("#", filled)) + dim(strings.Repeat("-", width-filled))
	}
	return fmt.Sprintf("[%s] %d/%d", bar, i, n)
}
