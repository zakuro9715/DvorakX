//go:build windows && e2e

package windows_test

import (
	"testing"

	"github.com/zakuro9715/DvorakZ/e2e/windows/env"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestWindowsRemap(t *testing.T) {
	exe := env.BuildRemapper(t)
	window := env.NewReceiver(t)

	// Confirm that Windows delivers unmodified input before installing the hook.
	env.AssertKey(t, window, keys.C, keys.C)

	env.StartRemapper(t, exe, "[global]\nremaps = [{ a = 'b' }]\n")

	// The receiver must see both edges of B without any original A events.
	env.AssertKey(t, window, keys.A, keys.B)

	// C has no mapping and must reach the receiver unchanged.
	env.AssertKey(t, window, keys.C, keys.C)
}
