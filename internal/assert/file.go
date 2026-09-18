package assert

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tianyuxue/clablings/internal/content"
)

// Sentinel is the completion marker string.
const Sentinel = "I AM NOT DONE"

func runFile(ctx context.Context, exec Executor, ex *content.Exercise, a *content.Assertion) Result {
	var contentStr string
	var source string

	if a.Node != "" && a.Path != "" {
		source = a.Node + ":" + a.Path
		res, err := exec.Exec(ctx, containerName(ex, a.Node), "cat "+a.Path)
		if err != nil {
			return Result{Passed: false, Expected: "readable " + source, Actual: "execution failed", Err: err}
		}
		if res.Code != 0 {
			return Result{Passed: false, Expected: "readable " + source, Actual: "read failed (code=" + strconv.Itoa(res.Code) + ")"}
		}
		contentStr = res.Stdout
	} else {
		source = a.Source
		data, err := os.ReadFile(filepath.Join(ex.Dir, a.Source))
		if err != nil {
			return Result{Passed: false, Expected: "readable " + source, Actual: "read failed", Err: err}
		}
		contentStr = string(data)
	}

	switch {
	case a.Contains != "":
		if strings.Contains(contentStr, a.Contains) {
			return Result{Passed: true, Expected: source + " contains " + strconv.Quote(a.Contains), Actual: "contains"}
		}
		return Result{Passed: false, Expected: source + " contains " + strconv.Quote(a.Contains), Actual: "does not contain"}
	case a.NotContains != "":
		if !strings.Contains(contentStr, a.NotContains) {
			return Result{Passed: true, Expected: source + " does not contain " + strconv.Quote(a.NotContains), Actual: "does not contain"}
		}
		return Result{Passed: false, Expected: source + " does not contain " + strconv.Quote(a.NotContains), Actual: "still contains"}
	default:
		if !strings.Contains(contentStr, Sentinel) {
			return Result{Passed: true, Expected: source + " has sentinel " + strconv.Quote(Sentinel) + " removed", Actual: "removed"}
		}
		return Result{Passed: false, Expected: source + " has sentinel " + strconv.Quote(Sentinel) + " removed", Actual: fmt.Sprintf("still contains %q", Sentinel)}
	}
}
