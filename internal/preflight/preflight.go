// Package preflight checks the external prerequisites clablings needs.
package preflight

import (
	"context"
	"fmt"
	"strings"

	"github.com/tianyuxue/clablings/internal/engine"
)

// Check verifies the docker daemon, containerlab, and required images,
// returning an aggregated error when anything is missing.
func Check(ctx context.Context, runner engine.Runner, clabBin, dockerBin string, images []string) error {
	var problems []string

	if res, err := runner.Run(ctx, dockerBin, "info"); err != nil {
		problems = append(problems, fmt.Sprintf("docker cannot be executed: %v", err))
	} else if res.Code != 0 {
		problems = append(problems, fmt.Sprintf("docker daemon is not running (docker info failed: %s)", strings.TrimSpace(res.Stderr)))
	}

	if res, err := runner.Run(ctx, clabBin, "version"); err != nil {
		problems = append(problems, fmt.Sprintf("containerlab cannot be executed: %v", err))
	} else if res.Code != 0 {
		problems = append(problems, fmt.Sprintf("containerlab is unavailable (containerlab version failed: %s)", strings.TrimSpace(res.Stderr)))
	}

	for _, img := range images {
		if img == "" {
			continue
		}
		res, err := runner.Run(ctx, dockerBin, "image", "inspect", img)
		if err != nil {
			problems = append(problems, fmt.Sprintf("cannot inspect image %s: %v", img, err))
		} else if res.Code != 0 {
			problems = append(problems, fmt.Sprintf("missing image %s (run `docker pull %s` first)", img, img))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("preflight failed:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}
