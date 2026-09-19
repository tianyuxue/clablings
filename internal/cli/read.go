package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/styles"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
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
					ch = chapterAt(lib, wd)
				}
				if ch == nil {
					if st, err := state.Load(resolved); err == nil && st.Current != "" {
						if ex := lib.ByID[st.Current]; ex != nil {
							ch = ex.ChapterRef
						}
					}
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

func printChapterRead(w io.Writer, ch *content.Chapter, loc string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: %s\n\n", ch.ID, ch.Title)
	if len(ch.Objectives) > 0 {
		b.WriteString("## Objectives\n\n")
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
	if err := renderMarkdown(w, b.String()); err != nil {
		return err
	}
	readings, err := content.LoadReading(ch.Dir, loc)
	if err != nil {
		return err
	}
	if len(readings) > 0 {
		if isTTY() {
			fmt.Fprintln(w, "\x1b[1mReading:\x1b[0m")
		} else {
			fmt.Fprintln(w, "Reading:")
		}
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
			fmt.Fprintf(w, "  - %s%s\n", hyperlink(label, r.URL), kind)
			if r.URL != "" {
				fmt.Fprintf(w, "      %s\n", r.URL)
			}
			if r.Intro != "" {
				fmt.Fprintf(w, "      Intro: %s\n", r.Intro)
			}
			if r.Question != "" {
				fmt.Fprintf(w, "      Question: %s\n", r.Question)
			}
		}
	}
	return nil
}

func isTTY() bool { return term.IsTerminal(int(os.Stdout.Fd())) }

// hyperlink renders text as an underlined, clickable OSC 8 hyperlink on a
// terminal; on a non-terminal it returns the plain text.
func hyperlink(text, url string) string {
	if url == "" || !isTTY() {
		return text
	}
	return "\x1b[4;36m\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\\x1b[0m"
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
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(st),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return err
	}
	out, err := r.Render(md)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, out)
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
