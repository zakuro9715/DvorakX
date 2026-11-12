package config

import "strings"

const (
	vkBack      = 0x08
	vkTab       = 0x09
	vkReturn    = 0x0D
	vkShift     = 0x10
	vkControl   = 0x11
	vkMenu      = 0x12
	vkCapital   = 0x14
	vkEscape    = 0x1B
	vkSpace     = 0x20
	vkLWin      = 0x5B
	vkRWin      = 0x5C
	vkNum0      = 0x30
	vkNum9      = 0x39
	vkA         = 0x41
	vkZ         = 0x5A
	vkLShift    = 0xA0
	vkRShift    = 0xA1
	vkLControl  = 0xA2
	vkRControl  = 0xA3
	vkLMenu     = 0xA4
	vkRMenu     = 0xA5
	vkOem1      = 0xBA
	vkOemPlus   = 0xBB
	vkOemComma  = 0xBC
	vkOemMinus  = 0xBD
	vkOemPeriod = 0xBE
	vkOem2      = 0xBF
	vkOem3      = 0xC0
	vkOem4      = 0xDB
	vkOem5      = 0xDC
	vkOem6      = 0xDD
	vkOem7      = 0xDE
)

var namedVirtualKeys = map[string]uint16{
	"bspc":  vkBack,
	"tab":   vkTab,
	"ret":   vkReturn,
	"enter": vkReturn,
	"caps":  vkCapital,
	"esc":   vkEscape,
	"spc":   vkSpace,
	"space": vkSpace,

	"shift": vkShift,
	"ctrl":  vkControl,
	"alt":   vkMenu,
	"met":   vkLWin,
	"lsft":  vkLShift,
	"rsft":  vkRShift,
	"lctl":  vkLControl,
	"rctl":  vkRControl,
	"lalt":  vkLMenu,
	"ralt":  vkRMenu,
	"lmet":  vkLWin,
	"rmet":  vkRWin,

	"grv":   vkOem3,
	"`":     vkOem3,
	"-":     vkOemMinus,
	"=":     vkOemPlus,
	"[":     vkOem4,
	"]":     vkOem6,
	"\\":    vkOem5,
	"|":     vkOem5,
	";":     vkOem1,
	"quote": vkOem7,
	"'":     vkOem7,
	",":     vkOemComma,
	".":     vkOemPeriod,
	"/":     vkOem2,
}

func virtualKey(name string) (uint16, bool) {
	name = strings.ToLower(name)
	if key, ok := namedVirtualKeys[name]; ok {
		return key, true
	}
	if len(name) != 1 {
		return 0, false
	}
	if name[0] >= '0' && name[0] <= '9' {
		return vkNum0 + uint16(name[0]-'0'), true
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		return vkA + uint16(name[0]-'a'), true
	}
	return 0, false
}
