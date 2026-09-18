package assert

import (
	"context"
	"time"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

const defaultTimeout = 30 * time.Second

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
		actx, cancel := context.WithTimeout(ctx, timeout)

		var res Result
		switch a.Type {
		case "exec":
			res = runExec(actx, exec, ex, a)
		case "file":
			res = runFile(actx, exec, ex, a)
		case "interface", "link":
			if !ifacesLoaded {
				ifaces, ifacesErr = exec.InspectInterfaces(actx, labName(ex))
				ifacesLoaded = true
			}
			if ifacesErr != nil {
				res = Result{Passed: false, Expected: "readable node interface state", Actual: "containerlab inspect interfaces failed", Err: ifacesErr}
			} else if a.Type == "interface" {
				res = runInterface(ex, a, ifaces)
			} else {
				res = runLink(ex, a, topo, ifaces)
			}
		default:
			res = Result{Passed: false, Actual: "unknown assertion type " + a.Type}
		}
		cancel()

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
