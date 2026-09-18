package assert

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/tianyuxue/clablings/internal/content"
)

func runExec(ctx context.Context, exec Executor, ex *content.Exercise, a *content.Assertion) Result {
	container := containerName(ex, a.Node)
	res, err := exec.Exec(ctx, container, a.Cmd)
	if err != nil {
		return Result{Passed: false, Expected: "command runs", Actual: "execution failed", Err: err}
	}

	var expectedParts []string
	var mismatches []string
	passed := true

	if a.Match != nil && a.Match.ExitCode != nil {
		expectedParts = append(expectedParts, "exit code = "+strconv.Itoa(*a.Match.ExitCode))
		if res.Code != *a.Match.ExitCode {
			passed = false
			mismatches = append(mismatches, fmt.Sprintf("exit code = %d", res.Code))
		}
	}

	if a.Match != nil && a.Match.JSON != "" {
		expectedParts = append(expectedParts, "stdout JSONPath "+a.Match.JSON+" "+describeTextMatch(a.Match.TextMatch))
		ok, actual, jerr := matchJSON(res.Stdout, a.Match.JSON, a.Match.TextMatch)
		if jerr != nil {
			return Result{Passed: false, Expected: strings.Join(expectedParts, "; "), Actual: jerr.Error(), Err: jerr}
		}
		if !ok {
			passed = false
			mismatches = append(mismatches, "JSONPath value = "+actual)
		}
	} else {
		tm := content.TextMatch{}
		if a.Match != nil && a.Match.Stdout != nil {
			tm = *a.Match.Stdout
		} else if a.Match != nil {
			tm = a.Match.TextMatch
		}
		if !textMatchEmpty(tm) {
			expectedParts = append(expectedParts, "stdout "+describeTextMatch(tm))
			ok, merr := matchText(res.Stdout, tm)
			if merr != nil {
				return Result{Passed: false, Expected: strings.Join(expectedParts, "; "), Actual: merr.Error(), Err: merr}
			}
			if !ok {
				passed = false
				mismatches = append(mismatches, "stdout = "+firstLine(res.Stdout))
			}
		}
	}

	// No explicit condition: the command must succeed (exit code 0).
	if len(expectedParts) == 0 {
		expectedParts = append(expectedParts, "command succeeds")
		if res.Code != 0 {
			passed = false
			mismatches = append(mismatches, fmt.Sprintf("exit code = %d", res.Code))
		}
	}

	actual := fmt.Sprintf("exit code = %d", res.Code)
	if len(mismatches) > 0 {
		actual = strings.Join(mismatches, "; ")
	} else if res.Stdout != "" {
		actual += "; stdout = " + firstLine(res.Stdout)
	}
	return Result{Passed: passed, Expected: strings.Join(expectedParts, "; "), Actual: actual}
}

func firstLine(s string) string {
	s = strings.TrimRight(s, "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strconv.Quote(s[:i]) + " ..."
	}
	return strconv.Quote(s)
}
