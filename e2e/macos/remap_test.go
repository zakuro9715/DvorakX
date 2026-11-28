//go:build darwin && cgo && e2e

package macos_test

import (
	"testing"

	"github.com/zakuro9715/DvorakZ/e2e/macos/env"
)

func TestMacOSRemap(t *testing.T) {
	const (
		keyA = 0x00 // Quartz A
		keyB = 0x0b // Quartz B
		keyC = 0x08 // Quartz C
	)
	exe := env.BuildRemapper(t)
	receiver := env.NewReceiver(t)

	// Confirm that macOS delivers both original edges before starting DvorakZ.
	receiver.AssertKey(t, keyC, keyC)

	env.StartRemapper(t, exe, "[global]\nremaps = [{ a = 'b' }]\n")

	// The downstream tap must receive both B edges and no original A edges.
	receiver.AssertKey(t, keyA, keyB)

	// C has no mapping and must reach the downstream tap unchanged.
	receiver.AssertKey(t, keyC, keyC)
}
