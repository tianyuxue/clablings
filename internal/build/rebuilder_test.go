package build

import (
	"testing"

	"github.com/tianyuxue/clablings/internal/reset"
)

func TestBuilderSatisfiesResetRebuilder(t *testing.T) {
	var _ reset.Rebuilder = New(nil, "docker", "gcc:13")
}
