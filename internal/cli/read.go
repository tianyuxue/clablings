package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

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
	fmt.Fprintf(w, "== %s: %s ==\n\n", ch.ID, ch.Title)
	if len(ch.Objectives) > 0 {
		fmt.Fprintln(w, "Objectives:")
		for _, o := range ch.Objectives {
			fmt.Fprintf(w, "  - %s\n", o)
		}
		fmt.Fprintln(w)
	}
	if concept := readConceptFile(ch.Dir, loc); concept == "" {
		fmt.Fprintln(w, "(no concept document)")
	} else {
		fmt.Fprint(w, concept)
		if !strings.HasSuffix(concept, "\n") {
			fmt.Fprintln(w)
		}
	}
	readings, err := content.LoadReading(ch.Dir, loc)
	if err != nil {
		return err
	}
	if len(readings) > 0 {
		fmt.Fprintln(w, "\nReading:")
		for _, r := range readings {
			kind := ""
			if r.Kind != "" {
				kind = " (" + r.Kind + ")"
			}
			section := ""
			if r.Section != "" {
				section = " §" + r.Section
			}
			if r.RFC > 0 {
				fmt.Fprintf(w, "- RFC %d%s %s%s\n", r.RFC, section, r.Title, kind)
			} else {
				fmt.Fprintf(w, "- %s%s%s\n", r.Title, section, kind)
			}
			if r.Intro != "" {
				fmt.Fprintf(w, "  Intro: %s\n", r.Intro)
			}
			if r.Question != "" {
				fmt.Fprintf(w, "  Question: %s\n", r.Question)
			}
			if r.URL != "" {
				fmt.Fprintf(w, "  %s\n", r.URL)
			}
		}
	}
	return nil
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
