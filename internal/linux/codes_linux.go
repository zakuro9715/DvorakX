//go:build amd64 || arm64

package linux

import "github.com/zakuro9715/DvorakZ/internal/keys"

// Ordinary set-1 key positions share their numeric value with evdev.
var specialCodes = map[string]uint16{
	"rctl": 97, "ralt": 100, "lmet": 125, "rmet": 126,
	"prtsc": 99, "home": 102, "up": 103, "pgup": 104,
	"left": 105, "right": 106, "end": 107, "down": 108,
	"pgdn": 109, "ins": 110, "del": 111, "menu": 139,
	"power": 116, "sleep": 142, "wake": 143, "kpenter": 96, "kp/": 98,
	"kana": 93, "hanja": 123, "hangul": 122, "intlro": 89,
	"henkan": 92, "muhenkan": 94, "yen": 124,
	"f13": 183, "f14": 184, "f15": 185, "f16": 186,
	"f17": 187, "f18": 188, "f19": 189, "f20": 190,
	"f21": 191, "f22": 192, "f23": 193, "f24": 194,
	"break": 223, "clear": 0x163, "pause": 119, "select": 0x161,
	"execute": 0x160, "help": 138, "zoom": 0x174,
	"browserback": 158, "browserforward": 159, "browserrefresh": 173,
	"browserstop": 128, "browsersearch": 217, "browserfavorites": 156, "browserhome": 172,
	"volumemute": 113, "volumedown": 114, "volumeup": 115,
	"medianext": 163, "mediaprev": 165, "mediastop": 166, "mediaplaypause": 164,
	"launchmail": 155, "launchmedia": 226, "launchapp1": 148, "launchapp2": 149,
}

var codeToKey, keyToCode = keyCodes()

func keyCodes() (map[uint16]keys.Key, map[keys.Key]uint16) {
	input := make(map[uint16]keys.Key)
	output := make(map[keys.Key]uint16)
	for code := uint16(1); code <= keys.F12; code++ {
		key := keys.Key{ScanCode: code}
		if canonical, ok := keys.Lookup(keys.Name(key)); ok && canonical == key {
			input[code], output[key] = key, code
		}
	}
	for name, code := range specialCodes {
		key, _ := keys.Lookup(name)
		input[code], output[key] = key, code
	}
	return input, output
}
