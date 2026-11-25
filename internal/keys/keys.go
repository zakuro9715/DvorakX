// Package keys defines configured key names and keyboard scan codes.
package keys

import (
	"fmt"
	"strings"
)

// Key identifies a keyboard scan code and whether it uses the extended prefix.
type Key struct {
	ScanCode uint16
	Extended bool
}

const (
	Escape         = 0x01
	Digit1         = 0x02
	Digit2         = 0x03
	Digit3         = 0x04
	Digit4         = 0x05
	Digit5         = 0x06
	Digit6         = 0x07
	Digit7         = 0x08
	Digit8         = 0x09
	Digit9         = 0x0A
	Digit0         = 0x0B
	Minus          = 0x0C
	Equal          = 0x0D
	Backspace      = 0x0E
	Tab            = 0x0F
	Q              = 0x10
	W              = 0x11
	E              = 0x12
	R              = 0x13
	T              = 0x14
	Y              = 0x15
	U              = 0x16
	I              = 0x17
	O              = 0x18
	P              = 0x19
	LeftBracket    = 0x1A
	RightBracket   = 0x1B
	Enter          = 0x1C
	LeftCtrl       = 0x1D
	A              = 0x1E
	S              = 0x1F
	D              = 0x20
	F              = 0x21
	G              = 0x22
	H              = 0x23
	J              = 0x24
	K              = 0x25
	L              = 0x26
	Semicolon      = 0x27
	Quote          = 0x28
	Grave          = 0x29
	LeftShift      = 0x2A
	Backslash      = 0x2B
	Z              = 0x2C
	X              = 0x2D
	C              = 0x2E
	V              = 0x2F
	B              = 0x30
	N              = 0x31
	M              = 0x32
	Comma          = 0x33
	Period         = 0x34
	Slash          = 0x35
	RightShift     = 0x36
	PrintScreen    = 0x37
	LeftAlt        = 0x38
	Space          = 0x39
	CapsLock       = 0x3A
	F1             = 0x3B
	F2             = 0x3C
	F3             = 0x3D
	F4             = 0x3E
	F5             = 0x3F
	F6             = 0x40
	F7             = 0x41
	F8             = 0x42
	F9             = 0x43
	F10            = 0x44
	NumLock        = 0x45
	ScrollLock     = 0x46
	Numpad7        = 0x47
	Numpad8        = 0x48
	Numpad9        = 0x49
	NumpadSubtract = 0x4A
	Numpad4        = 0x4B
	Numpad5        = 0x4C
	Numpad6        = 0x4D
	NumpadAdd      = 0x4E
	Numpad1        = 0x4F
	Numpad2        = 0x50
	Numpad3        = 0x51
	Numpad0        = 0x52
	NumpadDecimal  = 0x53
	NumpadMultiply = 0x37
	F11            = 0x57
	F12            = 0x58
	IntlBackslash  = 0x56
	RightCtrl      = 0x1D
	RightAlt       = 0x38
	LeftWin        = 0x5B
	RightWin       = 0x5C
	Kana           = 0x70
	Hanja          = 0x71
	Hangul         = 0x72
	IntlRo         = 0x73
	Henkan         = 0x79
	Muhenkan       = 0x7B
	Yen            = 0x7D
	F13            = 0x64
	F14            = 0x65
	F15            = 0x66
	F16            = 0x67
	F17            = 0x68
	F18            = 0x69
	F19            = 0x6A
	F20            = 0x6B
	F21            = 0x6C
	F22            = 0x6D
	F23            = 0x6E
	F24            = 0x76

	Home       = 0x47
	ArrowUp    = 0x48
	PageUp     = 0x49
	ArrowLeft  = 0x4B
	ArrowRight = 0x4D
	End        = 0x4F
	ArrowDown  = 0x50
	PageDown   = 0x51
	Insert     = 0x52
	Delete     = 0x53
	Menu       = 0x5D
	Power      = 0x5E
	Sleep      = 0x5F
	Wake       = 0x63
)

var canonical = map[Key]string{
	{ScanCode: Digit1}: "1",
	{ScanCode: Digit2}: "2",
	{ScanCode: Digit3}: "3",
	{ScanCode: Digit4}: "4",
	{ScanCode: Digit5}: "5",
	{ScanCode: Digit6}: "6",
	{ScanCode: Digit7}: "7",
	{ScanCode: Digit8}: "8",
	{ScanCode: Digit9}: "9",
	{ScanCode: Digit0}: "0",

	{ScanCode: A}: "a",
	{ScanCode: B}: "b",
	{ScanCode: C}: "c",
	{ScanCode: D}: "d",
	{ScanCode: E}: "e",
	{ScanCode: F}: "f",
	{ScanCode: G}: "g",
	{ScanCode: H}: "h",
	{ScanCode: I}: "i",
	{ScanCode: J}: "j",
	{ScanCode: K}: "k",
	{ScanCode: L}: "l",
	{ScanCode: M}: "m",
	{ScanCode: N}: "n",
	{ScanCode: O}: "o",
	{ScanCode: P}: "p",
	{ScanCode: Q}: "q",
	{ScanCode: R}: "r",
	{ScanCode: S}: "s",
	{ScanCode: T}: "t",
	{ScanCode: U}: "u",
	{ScanCode: V}: "v",
	{ScanCode: W}: "w",
	{ScanCode: X}: "x",
	{ScanCode: Y}: "y",
	{ScanCode: Z}: "z",

	{ScanCode: Backspace}: "bspc",
	{ScanCode: Tab}:       "tab",
	{ScanCode: Enter}:     "enter",
	{ScanCode: CapsLock}:  "caps",
	{ScanCode: Escape}:    "esc",
	{ScanCode: Space}:     "space",
	{ScanCode: LeftShift}: "shift",
	{ScanCode: LeftCtrl}:  "ctrl",
	{ScanCode: LeftAlt}:   "alt",

	{ScanCode: F1}:  "f1",
	{ScanCode: F2}:  "f2",
	{ScanCode: F3}:  "f3",
	{ScanCode: F4}:  "f4",
	{ScanCode: F5}:  "f5",
	{ScanCode: F6}:  "f6",
	{ScanCode: F7}:  "f7",
	{ScanCode: F8}:  "f8",
	{ScanCode: F9}:  "f9",
	{ScanCode: F10}: "f10",
	{ScanCode: F11}: "f11",
	{ScanCode: F12}: "f12",
	{ScanCode: F13}: "f13",
	{ScanCode: F14}: "f14",
	{ScanCode: F15}: "f15",
	{ScanCode: F16}: "f16",
	{ScanCode: F17}: "f17",
	{ScanCode: F18}: "f18",
	{ScanCode: F19}: "f19",
	{ScanCode: F20}: "f20",
	{ScanCode: F21}: "f21",
	{ScanCode: F22}: "f22",
	{ScanCode: F23}: "f23",
	{ScanCode: F24}: "f24",

	{ScanCode: NumLock}:        "numlock",
	{ScanCode: ScrollLock}:     "scroll",
	{ScanCode: Numpad7}:        "kp7",
	{ScanCode: Numpad8}:        "kp8",
	{ScanCode: Numpad9}:        "kp9",
	{ScanCode: NumpadSubtract}: "kp-",
	{ScanCode: NumpadMultiply}: "kp*",
	{ScanCode: Numpad4}:        "kp4",
	{ScanCode: Numpad5}:        "kp5",
	{ScanCode: Numpad6}:        "kp6",
	{ScanCode: NumpadAdd}:      "kp+",
	{ScanCode: Numpad1}:        "kp1",
	{ScanCode: Numpad2}:        "kp2",
	{ScanCode: Numpad3}:        "kp3",
	{ScanCode: Numpad0}:        "kp0",
	{ScanCode: NumpadDecimal}:  "kp.",

	{ScanCode: Grave}:         "grv",
	{ScanCode: Minus}:         "-",
	{ScanCode: Equal}:         "=",
	{ScanCode: LeftBracket}:   "[",
	{ScanCode: RightBracket}:  "]",
	{ScanCode: Backslash}:     "\\",
	{ScanCode: Semicolon}:     ";",
	{ScanCode: Quote}:         "quote",
	{ScanCode: Comma}:         ",",
	{ScanCode: Period}:        ".",
	{ScanCode: Slash}:         "/",
	{ScanCode: IntlBackslash}: "intlbackslash",
	{ScanCode: Kana}:          "kana",
	{ScanCode: Hanja}:         "hanja",
	{ScanCode: Hangul}:        "hangul",
	{ScanCode: IntlRo}:        "intlro",
	{ScanCode: Henkan}:        "henkan",
	{ScanCode: Muhenkan}:      "muhenkan",
	{ScanCode: Yen}:           "yen",

	{ScanCode: LeftCtrl, Extended: true}:    "rctl",
	{ScanCode: RightAlt, Extended: true}:    "ralt",
	{ScanCode: LeftWin, Extended: true}:     "lmet",
	{ScanCode: RightWin, Extended: true}:    "rmet",
	{ScanCode: PrintScreen, Extended: true}: "prtsc",
	{ScanCode: Home, Extended: true}:        "home",
	{ScanCode: ArrowUp, Extended: true}:     "up",
	{ScanCode: PageUp, Extended: true}:      "pgup",
	{ScanCode: ArrowLeft, Extended: true}:   "left",
	{ScanCode: ArrowRight, Extended: true}:  "right",
	{ScanCode: End, Extended: true}:         "end",
	{ScanCode: ArrowDown, Extended: true}:   "down",
	{ScanCode: PageDown, Extended: true}:    "pgdn",
	{ScanCode: Insert, Extended: true}:      "ins",
	{ScanCode: Delete, Extended: true}:      "del",
	{ScanCode: Menu, Extended: true}:        "menu",
	{ScanCode: Power, Extended: true}:       "power",
	{ScanCode: Sleep, Extended: true}:       "sleep",
	{ScanCode: Wake, Extended: true}:        "wake",
	{ScanCode: Enter, Extended: true}:       "kpenter",
	{ScanCode: Slash, Extended: true}:       "kp/",
}

// Lookup returns the scan code for a configured key name.
func Lookup(name string) (Key, bool) {
	name = strings.ToLower(name)
	if key, ok := named[name]; ok {
		return key, true
	}
	return Key{}, false
}

// Name returns a canonical configured name for a key when one exists.
func Name(key Key) string {
	if name, ok := canonical[key]; ok {
		return name
	}
	return fmt.Sprintf("0x%X", key.ScanCode)
}
