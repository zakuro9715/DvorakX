//go:build windows && !amd64 && !arm64

package windows

import "unsafe"

// input matches the 32-bit Windows INPUT structure. The final padding covers
// the portion of the INPUT union used by MOUSEINPUT but not KEYBDINPUT.
type input struct {
	Type uint32
	Ki   keyboardInput
	_    [8]byte
}

var (
	_ [unsafe.Sizeof(input{}) - 28]byte
	_ [28 - unsafe.Sizeof(input{})]byte
)
