//go:build linux && e2e

package linux_test

import (
	"testing"

	"github.com/zakuro9715/DvorakZ/e2e/linux/env"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestLinuxRemap(t *testing.T) {
	exe := env.BuildRemapper(t)
	injector, sourcePath, source := env.NewSource(t)

	// Verify the source delivers both edges before DvorakZ grabs it.
	env.AssertKeys(t, injector, source, -1, []uint16{keys.C}, []uint16{keys.C})

	env.StartRemapper(t, exe, sourcePath, "[global]\nremaps = [{ a = 'b' }]\n")
	output := env.OpenOutput(t)

	// A becomes B, C passes through, and the grabbed source stays silent.
	env.AssertKeys(t, injector, output, source, []uint16{keys.A, keys.C}, []uint16{keys.B, keys.C})
}
