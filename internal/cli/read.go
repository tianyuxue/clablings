package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	xansi "github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/content"
)

func newReadCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:          "read [<chapter_id>]",
		Short:        "Show a chapter's concept and RFC reading slices",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			loc := locale(cmd.ErrOrStderr())
			lib, err := content.LoadWithLocale(resolved, loc)
			if err != nil {
				return err
			}
			var ch *content.Chapter
			switch {
			case len(args) == 1:
				ch = lib.ChapterByID(args[0])
				if ch == nil {
					return fmt.Errorf("unknown chapter %q", args[0])
				}
			default:
				if wd, err := os.Getwd(); err == nil {
					ch = chapterForInvocation(lib, resolved, wd)
				}
				if ch == nil && len(lib.Chapters) > 0 {
					ch = lib.Chapters[0]
				}
			}
			if ch == nil {
				return fmt.Errorf("no chapter found")
			}
			return printChapterRead(cmd.OutOrStdout(), ch, loc)
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	return cmd
}

type readLabels struct {
	Objectives string
	Reading    string
	Intro      string
	Question   string
}

func labelsFor(loc string) readLabels {
	if content.NormalizeLocale(loc) == "zh" {
		return readLabels{Objectives: "目标", Reading: "阅读", Intro: "导读", Question: "问题"}
	}
	return readLabels{Objectives: "Objectives", Reading: "Reading", Intro: "Intro", Question: "Question"}
}

func printChapterRead(w io.Writer, ch *content.Chapter, loc string) error {
	labels := labelsFor(loc)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: %s\n\n", ch.ID, ch.Title)
	if len(ch.Objectives) > 0 {
		fmt.Fprintf(&b, "## %s\n\n", labels.Objectives)
		for _, o := range ch.Objectives {
			fmt.Fprintf(&b, "- %s\n", o)
		}
		b.WriteString("\n")
	}
	if concept := readConceptFile(ch.Dir, loc); concept != "" {
		b.WriteString(stripLeadingHeading(concept))
		if !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	readings, err := content.LoadReading(ch.Dir, loc)
	if err != nil {
		return err
	}
	if len(readings) > 0 {
		fmt.Fprintf(&b, "## %s\n\n", labels.Reading)
		for _, r := range readings {
			kind := ""
			if r.Kind != "" {
				kind = " (" + r.Kind + ")"
			}
			section := ""
			if r.Section != "" {
				section = " §" + r.Section
			}
			label := r.Title + section
			if r.RFC > 0 {
				label = fmt.Sprintf("RFC %d%s %s", r.RFC, section, r.Title)
			}
			fmt.Fprintf(&b, "- %s%s\n", label, kind)
			if r.URL != "" {
				fmt.Fprintf(&b, "  - %s\n", r.URL)
			}
			if r.Intro != "" {
				fmt.Fprintf(&b, "  - %s: %s\n", labels.Intro, r.Intro)
			}
			if r.Question != "" {
				fmt.Fprintf(&b, "  - %s: %s\n", labels.Question, r.Question)
			}
		}
	}
	return renderMarkdown(w, b.String())
}

// stripLeadingHeading drops a leading `# ...` title line from a concept body,
// since printChapterRead already emits the chapter title as the H1.
func stripLeadingHeading(md string) string {
	s := strings.TrimLeft(md, "\n")
	if strings.HasPrefix(s, "# ") {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[i+1:]
		} else {
			s = ""
		}
	}
	return strings.TrimLeft(s, "\n")
}

// spaceHanChars inserts a space between adjacent Han characters outside code
// spans and link destinations. Both glamour and x/ansi wrap at spaces; without
// these break opportunities a Chinese paragraph is one long "word".
func spaceHanChars(md string) string {
	var b strings.Builder
	inFence := false
	for _, line := range strings.SplitAfter(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			b.WriteString(line)
			continue
		}
		if inFence {
			b.WriteString(line)
			continue
		}
		b.WriteString(spaceHanLine(line))
	}
	return b.String()
}

func spaceHanLine(line string) string {
	var b strings.Builder
	inCode := false
	inURL := false
	prevHan := false
	prev := rune(0)
	for _, r := range line {
		if inURL {
			b.WriteRune(r)
			if r == ')' {
				inURL = false
			}
			continue
		}
		if r == '`' {
			inCode = !inCode
			b.WriteRune(r)
			prevHan = false
			prev = r
			continue
		}
		if inCode {
			b.WriteRune(r)
			prevHan = false
			prev = r
			continue
		}
		if r == '(' && prev == ']' {
			inURL = true
			b.WriteRune(r)
			prevHan = false
			prev = r
			continue
		}
		if unicode.Is(unicode.Han, r) {
			if prevHan {
				b.WriteByte(' ')
			}
			b.WriteRune(r)
			prevHan = true
		} else {
			b.WriteRune(r)
			prevHan = false
		}
		prev = r
	}
	return b.String()
}

// removeHanSpaces removes the break spaces inserted by spaceHanChars after the
// text has been wrapped. ANSI escape sequences are preserved and ignored when
// looking for adjacent Han characters.
func removeHanSpaces(s string) string {
	var b strings.Builder
	prevHan := false
	for i := 0; i < len(s); {
		if n, ok := ansiSequenceLen(s[i:]); ok {
			b.WriteString(s[i : i+n])
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == ' ' && prevHan && nextHan(s[i+size:]) {
			i += size
			continue
		}
		b.WriteString(s[i : i+size])
		prevHan = unicode.Is(unicode.Han, r)
		i += size
	}
	return b.String()
}

func ansiSequenceLen(s string) (int, bool) {
	if len(s) < 2 || s[0] != 0x1b || s[1] != '[' {
		return 0, false
	}
	for i := 2; i < len(s); i++ {
		if s[i] >= 0x40 && s[i] <= 0x7e {
			return i + 1, true
		}
	}
	return 0, false
}

func nextHan(s string) bool {
	for i := 0; i < len(s); {
		if n, ok := ansiSequenceLen(s[i:]); ok {
			i += n
			continue
		}
		r, _ := utf8.DecodeRuneInString(s[i:])
		return unicode.Is(unicode.Han, r)
	}
	return false
}

// wrapRendered re-wraps glamour's output at the terminal width. Glamour's
// word-wrap is not CJK-aware and can split tokens such as 802.1Q; x/ansi wraps
// by display width and keeps words intact. Indented code lines and URLs are
// left alone so commands and links stay copy-pasteable.
func wrapRendered(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = wrapRenderedLine(line, width)
	}
	return strings.Join(lines, "\n")
}

func wrapRenderedLine(line string, width int) string {
	plain := xansi.Strip(line)
	if xansi.StringWidthWc(plain) <= width {
		return line
	}
	if strings.Contains(plain, "http://") || strings.Contains(plain, "https://") {
		return line
	}
	indent := len(plain) - len(strings.TrimLeft(plain, " "))
	if indent >= 4 {
		return line
	}
	parts := strings.Split(xansi.WrapWc(line, width, ""), "\n")
	if len(parts) <= 1 {
		return line
	}
	prefix := strings.Repeat(" ", indent)
	for i := 1; i < len(parts); i++ {
		parts[i] = prefix + parts[i]
	}
	return strings.Join(parts, "\n")
}

// renderMarkdown renders Markdown for a terminal: styled when stdout is a TTY,
// raw Markdown otherwise (pipes/CI). Width follows the terminal, defaulting to 80.
func renderMarkdown(w io.Writer, md string) error {
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		_, err := io.WriteString(w, md)
		return err
	}
	width := 80
	if cols, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && cols > 0 {
		width = cols
	}
	st := styles.DraculaStyleConfig
	st.H1.Prefix, st.H2.Prefix, st.H3.Prefix = "", "", ""
	st.H4.Prefix, st.H5.Prefix, st.H6.Prefix = "", "", ""
	// Let x/ansi do the wrapping: glamour's wrapper is not CJK-aware.
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(st),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		return err
	}
	out, err := r.Render(spaceHanChars(md))
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, removeHanSpaces(wrapRendered(out, width)))
	return err
}

// readConceptFile returns concept.<locale>.md with fallback to the default
// language, then concept.md.
func readConceptFile(dir, loc string) string {
	for _, name := range []string{"concept." + loc + ".md", "concept." + content.DefaultLocale + ".md", "concept.md"} {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return string(b)
		}
	}
	return ""
}
