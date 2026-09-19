package assert

import (
	"context"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

const defaultTimeout = 30 * time.Second
const pollInterval = 500 * time.Millisecond

// RunExercise runs all assertions of the exercise. topo is only needed by link
// assertions and may be nil.
func RunExercise(ctx context.Context, exec Executor, topo *engine.Topology, ex *content.Exercise) []Result {
	return RunFiltered(ctx, exec, topo, ex, nil)
}

// RunFiltered runs only the assertions for which keep returns true; a nil keep
// runs all assertions.
func RunFiltered(ctx context.Context, exec Executor, topo *engine.Topology, ex *content.Exercise, keep func(content.Assertion) bool) []Result {
	results := make([]Result, 0, len(ex.Assertions))
	var ifaces map[string][]engine.InspectInterface
	var ifacesErr error
	ifacesLoaded := false

	loadIfaces := func(c context.Context) {
		if !ifacesLoaded {
			ifaces, ifacesErr = exec.InspectInterfaces(c, labName(ex))
			ifacesLoaded = true
		}
	}

	for i := range ex.Assertions {
		a := &ex.Assertions[i]
		if keep != nil && !keep(*a) {
			continue
		}
		timeout := defaultTimeout
		if d, err := time.ParseDuration(a.Timeout); err == nil && d > 0 {
			timeout = d
		} else if d, err := time.ParseDuration(ex.Timeout); err == nil && d > 0 {
			timeout = d
		}
		wait := time.Duration(0)
		if d, err := time.ParseDuration(a.Wait); err == nil && d > 0 {
			wait = d
		}

		runOnce := func(reload bool) Result {
			if reload {
				ifacesLoaded = false
			}
			actx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()
			switch a.Type {
			case "exec":
				return runExec(actx, exec, ex, a)
			case "file":
				return runFile(actx, exec, ex, a)
			case "interface", "link":
				loadIfaces(actx)
				if ifacesErr != nil {
					return Result{Passed: false, Expected: "readable node interface state", Actual: "containerlab inspect interfaces failed", Err: ifacesErr}
				}
				if a.Type == "interface" {
					return runInterface(ex, a, ifaces)
				}
				return runLink(ex, a, topo, ifaces)
			default:
				return Result{Passed: false, Actual: "unknown assertion type " + a.Type}
			}
		}

		res := runOnce(false)
		if wait > 0 {
			deadline := time.Now().Add(wait)
			for !res.Passed && time.Now().Before(deadline) {
				time.Sleep(pollInterval)
				res = runOnce(true)
			}
		}

		res.Exercise = ex.FullID()
		res.Name = a.Name
		res.Type = a.Type
		res.Layer = a.Layer
		res.Viewpoint = a.Node
		res.Hint = a.Hint
		results = append(results, res)
	}
	return results
}
