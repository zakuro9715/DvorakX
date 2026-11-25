package examples

import (
	"testing"

	"github.com/zakuro9715/DvorakZ/testing/env"
)

func TestAllBehavior(t *testing.T) {
	input := `
# Layer remaps, direct global remaps, noop actions, and send-keys actions.
a b c d caps met x y z
`
	output := `
b c d ctrl alt
ctrl:down c v ctrl:up
ctrl:down c v ctrl:up
`
	env.LoadConfig(t, "all.toml").
		Test(input, output)
}
