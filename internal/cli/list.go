package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/state"
)

func newListCmd() *cobra.Command {
	var root string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List chapters, exercises, and progress",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			lib, err := content.Load(resolved)
			if err != nil {
				return err
			}
			st, err := state.Load(resolved)
			if err != nil {
				var corrupt *state.CorruptError
				if !errors.As(err, &corrupt) {
					return err
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			}
			if asJSON {
				return writeJSON(cmd, lib, st)
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), formatLibraryText(lib, st))
			return err
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root directory (defaults to searching upward for clablings.yaml)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func statusMarker(s state.Status) string {
	switch s {
	case state.StatusPassed:
		return "✓"
	case state.StatusInProgress:
		return "▸"
	default:
		return "·"
	}
}

func formatLibraryText(lib *content.Library, st *state.State) string {
	var b strings.Builder
	for _, ch := range lib.Chapters {
		fmt.Fprintf(&b, "%s %s\n", ch.ID, ch.Title)
		for i := range ch.Exercises {
			ex := ch.Exercises[i]
			es := st.Get(ex.FullID())
			fmt.Fprintf(&b, "  %s %s %s\n", statusMarker(es.Status), ex.ID, ex.Title)
		}
	}
	return b.String()
}

type jsonExercise struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Attempts int    `json:"attempts"`
}

type jsonChapter struct {
	ID        string         `json:"id"`
	Title     string         `json:"title"`
	Exercises []jsonExercise `json:"exercises"`
}

func writeJSON(cmd *cobra.Command, lib *content.Library, st *state.State) error {
	view := struct {
		Root     string        `json:"root"`
		Current  string        `json:"current"`
		Chapters []jsonChapter `json:"chapters"`
	}{Root: lib.Root, Current: st.Current}

	for _, ch := range lib.Chapters {
		jc := jsonChapter{ID: ch.ID, Title: ch.Title}
		for i := range ch.Exercises {
			ex := ch.Exercises[i]
			es := st.Get(ex.FullID())
			jc.Exercises = append(jc.Exercises, jsonExercise{
				ID:       ex.ID,
				Title:    ex.Title,
				Type:     string(ex.Type),
				Status:   string(es.Status),
				Attempts: es.Attempts,
			})
		}
		view.Chapters = append(view.Chapters, jc)
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(view)
}
