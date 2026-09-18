package reset

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/engine"
)

func TestRealEngineSatisfiesResetEngine(t *testing.T) {
	var _ Engine = FromEngine(engine.New("containerlab", "docker"))
}
