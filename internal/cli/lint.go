package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

func newLintCmd() *cobra.Command {
	var root string
	cmd := &cobra.Command{
		Use:          "lint",
		Short:        "Validate the exercise library",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			problems, err := lintLibrary(resolved)
			if err != nil {
				return err
			}
			warns, err := content.MissingTranslations(resolved, "zh")
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			for _, p := range problems {
				fmt.Fprintln(out, "error: "+p)
			}
			for _, w := range warns {
				fmt.Fprintln(out, "warning: "+w)
			}
			if len(problems) > 0 {
				fmt.Fprintf(out, "%d problem(s), %d warning(s)\n", len(problems), len(warns))
				return &exitCodeError{code: 1}
			}
			fmt.Fprintf(out, "ok: %d warning(s)\n", len(warns))
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	return cmd
}

func lintLibrary(root string) ([]string, error) {
	lib, err := content.LoadWithLocale(root, content.DefaultLocale)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, ch := range lib.Chapters {
		if ch.HasTopology() {
			topo, err := engine.ParseTopology(ch.TopologyPath)
			if err != nil {
				return nil, err
			}
			nodes := map[string]bool{}
			for _, n := range topo.Nodes {
				nodes[n.Name] = true
			}
			checkNode := func(exID string, idx int, kind, node string) {
				if node == "" {
					problems = append(problems, fmt.Sprintf("%s: %s %d requires a node", exID, kind, idx))
					return
				}
				if !nodes[node] {
					problems = append(problems, fmt.Sprintf("%s: %s %d references unknown node %q", exID, kind, idx, node))
				}
			}
			for _, ex := range ch.Exercises {
				for i, a := range ex.Assertions {
					switch a.Type {
					case "exec", "interface":
						checkNode(ex.FullID(), i+1, "assertion", a.Node)
					case "link":
						checkNode(ex.FullID(), i+1, "link endpoint a", a.A)
						checkNode(ex.FullID(), i+1, "link endpoint b", a.B)
					case "file":
						if a.Node != "" && !nodes[a.Node] {
							problems = append(problems, fmt.Sprintf("%s: assertion %d references unknown node %q", ex.FullID(), i+1, a.Node))
						}
					default:
						problems = append(problems, fmt.Sprintf("%s: assertion %d has unknown type %q", ex.FullID(), i+1, a.Type))
					}
				}
				for _, f := range ex.Files {
					if f.Node != "" && !nodes[f.Node] {
						problems = append(problems, fmt.Sprintf("%s: file %q references unknown node %q", ex.FullID(), f.Source, f.Node))
					}
				}
				for _, r := range ex.Reload {
					if r.Node != "" && !nodes[r.Node] {
						problems = append(problems, fmt.Sprintf("%s: reload references unknown node %q", ex.FullID(), r.Node))
					}
				}
				if ex.Run != nil && ex.Run.Node != "" && !nodes[ex.Run.Node] {
					problems = append(problems, fmt.Sprintf("%s: run references unknown node %q", ex.FullID(), ex.Run.Node))
				}
			}
		} else {
			for _, ex := range ch.Exercises {
				for i, a := range ex.Assertions {
					if a.Type == "exec" || a.Type == "interface" || a.Type == "link" || a.Node != "" || a.Path != "" {
						problems = append(problems, fmt.Sprintf("%s: assertion %d requires a topology", ex.FullID(), i+1))
					}
				}
				if len(ex.Reload) > 0 || ex.Build != nil || ex.Run != nil {
					problems = append(problems, fmt.Sprintf("%s: reload/build/run require a topology", ex.FullID()))
				}
				for _, f := range ex.Files {
					if f.Node != "" || f.Dest != "" {
						problems = append(problems, fmt.Sprintf("%s: file %q targets a node but the chapter has no topology", ex.FullID(), f.Source))
					}
				}
			}
		}
		for _, ex := range ch.Exercises {
			for i, a := range ex.Assertions {
				if a.Wait != "" {
					if d, err := time.ParseDuration(a.Wait); err != nil || d < 0 {
						problems = append(problems, fmt.Sprintf("%s: assertion %d has invalid wait %q", ex.FullID(), i+1, a.Wait))
					}
				}
			}
		}
	}
	return problems, nil
}
