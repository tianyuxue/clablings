package cli

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/engine"
	"github.com/tianyuxue/clablings/internal/watch"
)

func TestLifecycleAdapterSatisfiesWatchEngine(t *testing.T) {
	var _ watch.Engine = newLifecycleAdapter(engine.New("containerlab", "docker"))
}
