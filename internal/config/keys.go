package config

import "strings"

const (
	scEscape    = 0x01
	scBack      = 0x0E
	scTab       = 0x0F
	scReturn    = 0x1C
	scSpace     = 0x39
	scCapital   = 0x3A
	scShift     = 0x2A
	scControl   = 0x1D
	scMenu      = 0x38
	scNum1      = 0x02
	scNum2      = 0x03
	scNum3      = 0x04
	scNum4      = 0x05
	scNum5      = 0x06
	scNum6      = 0x07
	scNum7      = 0x08
	scNum8      = 0x09
	scNum9      = 0x0A
	scNum0      = 0x0B
	scA         = 0x1E
	scB         = 0x30
	scC         = 0x2E
	scD         = 0x20
	scE         = 0x12
	scF         = 0x21
	scG         = 0x22
	scH         = 0x23
	scI         = 0x17
	scJ         = 0x24
	scK         = 0x25
	scL         = 0x26
	scM         = 0x32
	scN         = 0x31
	scO         = 0x18
	scP         = 0x19
	scQ         = 0x10
	scR         = 0x13
	scS         = 0x1F
	scT         = 0x14
	scU         = 0x16
	scV         = 0x2F
	scW         = 0x11
	scX         = 0x2D
	scY         = 0x15
	scZ         = 0x2C
	scLShift    = 0x2A
	scRShift    = 0x36
	scLControl  = 0x1D
	scRControl  = 0xE01D
	scLMenu     = 0x38
	scRMenu     = 0xE038
	scLWin      = 0xE05B
	scRWin      = 0xE05C
	scOem1      = 0x27
	scOemPlus   = 0x0D
	scOemComma  = 0x33
	scOemMinus  = 0x0C
	scOemPeriod = 0x34
	scOem2      = 0x35
	scOem3      = 0x29
	scOem4      = 0x1A
	scOem5      = 0x2B
	scOem6      = 0x1B
	scOem7      = 0x28
)

var namedScanCodes = map[string]uint16{
	"bspc":  0x0E,
	"tab":   0x0F,
	"ret":   0x1C,
	"enter": 0x1C,
	"caps":  0x3A,
	"esc":   0x01,
	"spc":   0x39,
	"space": 0x39,

	"shift": 0x2A,
	"ctrl":  0x1D,
	"alt":   0x38,
	"met":   0xE05B,
	"lsft":  0x2A,
	"rsft":  0x36,
	"lctl":  0x1D,
	"rctl":  0xE01D,
	"lalt":  0x38,
	"ralt":  0xE038,
	"lmet":  0xE05B,
	"rmet":  0xE05C,

	"grv":   0x29,
	"`":     0x29,
	"-":     0x0C,
	"=":     0x0D,
	"[":     0x1A,
	"]":     0x1B,
	"\\":    0x2B,
	"|":     0x2B,
	";":     0x27,
	"quote": 0x28,
	"'":     0x28,
	",":     0x33,
	".":     0x34,
	"/":     0x35,
}

func scanCode(name string) (uint16, bool) {
	name = strings.ToLower(name)
	if code, ok := namedScanCodes[name]; ok {
		return code, true
	}
	if len(name) != 1 {
		return 0, false
	}
	if name[0] >= '0' && name[0] <= '9' {
		if name[0] == '0' {
			return scNum0, true
		}
		return scNum1 + uint16(name[0]-'1'), true
	}
	if name[0] >= 'a' && name[0] <= 'z' {
		switch name[0] {
		case 'a':
			return scA, true
		case 'b':
			return scB, true
		case 'c':
			return scC, true
		case 'd':
			return scD, true
		case 'e':
			return scE, true
		case 'f':
			return scF, true
		case 'g':
			return scG, true
		case 'h':
			return scH, true
		case 'i':
			return scI, true
		case 'j':
			return scJ, true
		case 'k':
			return scK, true
		case 'l':
			return scL, true
		case 'm':
			return scM, true
		case 'n':
			return scN, true
		case 'o':
			return scO, true
		case 'p':
			return scP, true
		case 'q':
			return scQ, true
		case 'r':
			return scR, true
		case 's':
			return scS, true
		case 't':
			return scT, true
		case 'u':
			return scU, true
		case 'v':
			return scV, true
		case 'w':
			return scW, true
		case 'x':
			return scX, true
		case 'y':
			return scY, true
		case 'z':
			return scZ, true
		}
	}
	return 0, false
}
