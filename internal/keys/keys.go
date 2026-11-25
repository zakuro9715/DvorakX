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
	Escape    = 0x01
	Num1      = 0x02
	Num0      = 0x0B
	Minus     = 0x0C
	Equal     = 0x0D
	Back      = 0x0E
	Tab       = 0x0F
	Q         = 0x10
	P         = 0x19
	LBrack    = 0x1A
	RBrack    = 0x1B
	Return    = 0x1C
	Lctrl     = 0x1D
	A         = 0x1E
	L         = 0x26
	Semi      = 0x27
	Quote     = 0x28
	Grv       = 0x29
	Lshift    = 0x2A
	Backslash = 0x2B
	Z         = 0x2C
	M         = 0x32
	Comma     = 0x33
	Period    = 0x34
	Slash     = 0x35
	Rshift    = 0x36
	Lalt      = 0x38
	Space     = 0x39
	Caps      = 0x3A
	Rctrl     = 0x1D
	Ralt      = 0x38
	Lwin      = 0x5B
	Rwin      = 0x5C
)

var named = map[string]Key{
	"bspc":  {ScanCode: Back},
	"tab":   {ScanCode: Tab},
	"ret":   {ScanCode: Return},
	"enter": {ScanCode: Return},
	"caps":  {ScanCode: Caps},
	"esc":   {ScanCode: Escape},
	"spc":   {ScanCode: Space},
	"space": {ScanCode: Space},

	"shift": {ScanCode: Lshift},
	"ctrl":  {ScanCode: Lctrl},
	"alt":   {ScanCode: Lalt},
	"met":   {ScanCode: Lwin, Extended: true},
	"lsft":  {ScanCode: Lshift},
	"rsft":  {ScanCode: Rshift},
	"lctl":  {ScanCode: Lctrl},
	"rctl":  {ScanCode: Rctrl, Extended: true},
	"lalt":  {ScanCode: Lalt},
	"ralt":  {ScanCode: Ralt, Extended: true},
	"lmet":  {ScanCode: Lwin, Extended: true},
	"rmet":  {ScanCode: Rwin, Extended: true},

	"grv":   {ScanCode: Grv},
	"`":     {ScanCode: Grv},
	"-":     {ScanCode: Minus},
	"=":     {ScanCode: Equal},
	"[":     {ScanCode: LBrack},
	"]":     {ScanCode: RBrack},
	"\\":    {ScanCode: Backslash},
	"|":     {ScanCode: Backslash},
	";":     {ScanCode: Semi},
	"quote": {ScanCode: Quote},
	"'":     {ScanCode: Quote},
	",":     {ScanCode: Comma},
	".":     {ScanCode: Period},
	"/":     {ScanCode: Slash},
}

// Lookup returns the scan code for a configured key name.
func Lookup(name string) (Key, bool) {
	name = strings.ToLower(name)
	if key, ok := named[name]; ok {
		return key, true
	}
	if len(name) != 1 {
		return Key{}, false
	}
	if name[0] >= '0' && name[0] <= '9' {
		return Key{ScanCode: digitScan[name[0]-'0']}, true
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		return Key{ScanCode: letterScan[name[0]-'a']}, true
	}
	return Key{}, false
}

// Name returns a canonical configured name for a key when one exists.
func Name(key Key) string {
	if key.Extended {
		switch key.ScanCode {
		case Lctrl:
			return "rctl"
		case Ralt:
			return "ralt"
		case Lwin:
			return "lmet"
		case Rwin:
			return "rmet"
		}
		return ""
	}
	switch key.ScanCode {
	case Back:
		return "bspc"
	case Tab:
		return "tab"
	case Return:
		return "enter"
	case Caps:
		return "caps"
	case Escape:
		return "esc"
	case Space:
		return "space"
	case Lshift:
		return "shift"
	case Lctrl:
		return "ctrl"
	case Lalt:
		return "alt"
	}
	for name, candidate := range named {
		if candidate == key {
			return name
		}
	}
	for index, scanCode := range digitScan {
		if scanCode == key.ScanCode {
			return string(rune('0' + index))
		}
	}
	for index, scanCode := range letterScan {
		if scanCode == key.ScanCode {
			return string(rune('a' + index))
		}
	}
	return fmt.Sprintf("0x%X", key.ScanCode)
}

var (
	digitScan  = [10]uint16{0x0B, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A}
	letterScan = [26]uint16{
		0x1E, 0x30, 0x2E, 0x20, 0x12, 0x21, 0x22, 0x23, 0x17, 0x24, 0x25, 0x26,
		0x31, 0x32, 0x18, 0x19, 0x10, 0x13, 0x1F, 0x14, 0x16, 0x2F, 0x11, 0x2D,
		0x15, 0x2C,
	}
)
