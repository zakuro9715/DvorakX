//go:build windows && (amd64 || arm64)

package windows

import "unsafe"

// input matches the 64-bit Windows INPUT structure. The final padding covers
// the portion of the INPUT union used by MOUSEINPUT but not KEYBDINPUT.
type input struct {
	Type uint32
	_    uint32
	Ki   keyboardInput
	_    [8]byte
}

var _ [unsafe.Sizeof(input{}) - 40]byte
var _ [40 - unsafe.Sizeof(input{})]byte
