package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tianyuxue/clablings/internal/assert"
	"github.com/tianyuxue/clablings/internal/build"
	"github.com/tianyuxue/clablings/internal/config"
	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/headless"
	"github.com/tianyuxue/clablings/internal/preflight"
	"github.com/tianyuxue/clablings/internal/report"
	"github.com/tianyuxue/clablings/internal/reset"
)

func newTestCmd() *cobra.Command {
	var root string
	var all bool
	var format string
	var answersPath string
	cmd := &cobra.Command{
		Use:   "test [<exercise_id>]",
		Short: "Headless exercise verification for CI",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseTestArgs(args, all)
			if err != nil {
				return err
			}
			resolved, err := resolveRoot(root)
			if err != nil {
				return err
			}
			code, err := runHeadless(cmd.Context(), resolved, id, all, answersPath, format, cmd)
			if err != nil {
				return err
			}
			if code != 0 {
				return &exitCodeError{code: code}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&root, "root", "", "library root (default: search upward for clablings.yaml)")
	cmd.Flags().BoolVar(&all, "all", false, "verify every exercise in the library")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&answersPath, "answers", "", "YAML answers file for quiz exercises")
	cmd.SilenceUsage = true
	return cmd
}

func parseTestArgs(args []string, all bool) (string, error) {
	if all {
		if len(args) > 0 {
			return "", errors.New("--all cannot be combined with an exercise ID")
		}
		return "", nil
	}
	if len(args) != 1 {
		return "", errors.New("an exercise ID is required, or use --all")
	}
	return args[0], nil
}

func exitCodeForResults(results []assert.Result) int {
	for _, r := range results {
		if !r.Passed {
			return 1
		}
	}
	return 0
}

// unique drops empty strings and duplicates, preserving order.
func unique(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func runHeadless(ctx context.Context, root, id string, all bool, answersPath, format string, cmd *cobra.Command) (int, error) {
	lib, err := content.LoadWithLocale(root, locale(cmd.ErrOrStderr()))
	if err != nil {
		return 2, err
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: reading config failed, using defaults: %v\n", err)
	}
	clabBin := binOr(cfg.Containerlab.BinPath, "containerlab")
	dockerBin := binOr(cfg.Docker.BinPath, "docker")
	eng := engine.New(clabBin, dockerBin)
	images := []string{}
	for _, ex := range lib.Order {
		if ex.Build != nil && ex.Build.Image != "" {
			images = append(images, ex.Build.Image)
		}
		if ex.ChapterRef != nil && ex.ChapterRef.HasTopology() {
			if topo, terr := engine.ParseTopology(ex.ChapterRef.TopologyPath); terr == nil {
				for _, n := range topo.Nodes {
					if n.Image != "" {
						images = append(images, n.Image)
					}
				}
			}
		}
	}
	if err := preflight.Check(ctx, eng.Runner(), clabBin, dockerBin, unique(images)); err != nil {
		return 2, err
	}

	builder := build.New(eng.Runner(), dockerBin, cfg.Build.DefaultImage)
	res := reset.New(reset.Options{Root: root, Engine: reset.FromEngine(eng), Rebuilder: builder})
	answers := content.Answers{}
	if answersPath != "" {
		answers, err = content.LoadAnswers(answersPath)
		if err != nil {
			return 2, err
		}
	}
	baseVerify := makeVerifier(eng, builder)
	verify := func(ctx context.Context, ex *content.Exercise) ([]assert.Result, error) {
		if ex.Type == content.TypeQuiz {
			if ans, ok := answers[ex.FullID()]; ok {
				return assert.GradeQuiz(ex, ans), nil
			}
			return nil, nil
		}
		return baseVerify(ctx, ex)
	}
	skip := func(ex *content.Exercise) bool {
		return ex.Type == content.TypeQuiz && answers[ex.FullID()] == nil
	}

	failed := false
	var allResults []assert.Result
	emit := func(exID string, results []assert.Result) {
		if exitCodeForResults(results) != 0 {
			failed = true
		}
		allResults = append(allResults, results...)
		if format == "json" {
			return
		}
		fmt.Fprintf(cmd.OutOrStdout(), "== %s ==\n", exID)
		report.Text(cmd.OutOrStdout(), results)
	}

	if all {
		if err := headless.RunAll(ctx, lib, newLifecycleAdapter(eng), res, verify, skip, emit); err != nil {
			return 2, err
		}
	} else {
		results, err := headless.RunOne(ctx, lib, id, newLifecycleAdapter(eng), res, verify, skip)
		if err != nil {
			return 2, err
		}
		emit(id, results)
	}

	if format == "json" {
		if err := report.JSON(cmd.OutOrStdout(), allResults); err != nil {
			return 2, err
		}
	}
	if failed {
		return 1, nil
	}
	return 0, nil
}
