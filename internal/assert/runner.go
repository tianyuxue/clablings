package assert

import (
	"context"

	"github.com/tianyuxue/clablings/internal/content"
	"github.com/tianyuxue/clablings/internal/engine"
)

// Executor is the minimal engine capability assertions need. *engine.Engine satisfies it.
type Executor interface {
	Exec(ctx context.Context, container, cmd string) (engine.Result, error)
	InspectInterfaces(ctx context.Context, labName string) (map[string][]engine.InspectInterface, error)
}

// Compile-time check that the real engine satisfies Executor.
var _ Executor = (*engine.Engine)(nil)

// Result is the structured outcome of one assertion.
type Result struct {
	Exercise  string
	Name      string
	Type      string
	Layer     string
	Viewpoint string
	Expected  string
	Actual    string
	Hint      string
	Passed    bool
	Err       error
}

// containerName derives the container long name from the chapter topology name.
func containerName(ex *content.Exercise, node string) string {
	if ex.ChapterRef != nil && ex.ChapterRef.TopologyName != "" {
		return "clab-" + ex.ChapterRef.TopologyName + "-" + node
	}
	return node
}

func labName(ex *content.Exercise) string {
	if ex.ChapterRef != nil {
		return ex.ChapterRef.TopologyName
	}
	return ""
}
