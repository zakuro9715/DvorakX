// Package keys defines configured key names and Windows virtual-key codes.
package keys

import "strings"

const (
	VKBack      = 0x08
	VKTab       = 0x09
	VKReturn    = 0x0D
	VKShift     = 0x10
	VKControl   = 0x11
	VKMenu      = 0x12
	VKCapital   = 0x14
	VKEscape    = 0x1B
	VKSpace     = 0x20
	VKLWin      = 0x5B
	VKRWin      = 0x5C
	VKNum0      = 0x30
	VKNum9      = 0x39
	VKA         = 0x41
	VKZ         = 0x5A
	VKLShift    = 0xA0
	VKRShift    = 0xA1
	VKLControl  = 0xA2
	VKRControl  = 0xA3
	VKLMenu     = 0xA4
	VKRMenu     = 0xA5
	VKOEM1      = 0xBA
	VKOEMPlus   = 0xBB
	VKOEMComma  = 0xBC
	VKOEMMinus  = 0xBD
	VKOEMPeriod = 0xBE
	VKOEM2      = 0xBF
	VKOEM3      = 0xC0
	VKOEM4      = 0xDB
	VKOEM5      = 0xDC
	VKOEM6      = 0xDD
	VKOEM7      = 0xDE
)

var named = map[string]uint16{
	"bspc":  VKBack,
	"tab":   VKTab,
	"ret":   VKReturn,
	"enter": VKReturn,
	"caps":  VKCapital,
	"esc":   VKEscape,
	"spc":   VKSpace,
	"space": VKSpace,

	"shift": VKShift,
	"ctrl":  VKControl,
	"alt":   VKMenu,
	"met":   VKLWin,
	"lsft":  VKLShift,
	"rsft":  VKRShift,
	"lctl":  VKLControl,
	"rctl":  VKRControl,
	"lalt":  VKLMenu,
	"ralt":  VKRMenu,
	"lmet":  VKLWin,
	"rmet":  VKRWin,

	"grv":   VKOEM3,
	"`":     VKOEM3,
	"-":     VKOEMMinus,
	"=":     VKOEMPlus,
	"[":     VKOEM4,
	"]":     VKOEM6,
	"\\":    VKOEM5,
	"|":     VKOEM5,
	";":     VKOEM1,
	"quote": VKOEM7,
	"'":     VKOEM7,
	",":     VKOEMComma,
	".":     VKOEMPeriod,
	"/":     VKOEM2,
}

// Lookup returns the virtual-key code for a configured key name.
func Lookup(name string) (uint16, bool) {
	name = strings.ToLower(name)
	if key, ok := named[name]; ok {
		return key, true
	}
	if len(name) != 1 {
		return 0, false
	}
	if name[0] >= '0' && name[0] <= '9' {
		return VKNum0 + uint16(name[0]-'0'), true
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		return VKA + uint16(name[0]-'a'), true
	}
	return 0, false
}
