//go:build darwin

package macos

import (
	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

const mediaBase = 256

// Quartz keyboard codes are indexed by physical position.
var names = map[int]string{
	0: "a", 1: "s", 2: "d", 3: "f", 4: "h", 5: "g", 6: "z", 7: "x",
	8: "c", 9: "v", 10: "intlbackslash", 11: "b", 12: "q", 13: "w", 14: "e", 15: "r",
	16: "y", 17: "t", 18: "1", 19: "2", 20: "3", 21: "4", 22: "6", 23: "5",
	24: "=", 25: "9", 26: "7", 27: "-", 28: "8", 29: "0", 30: "]", 31: "o",
	32: "u", 33: "[", 34: "i", 35: "p", 36: "enter", 37: "l", 38: "j", 39: "quote",
	40: "k", 41: ";", 42: "\\", 43: ",", 44: "/", 45: "n", 46: "m", 47: ".",
	48: "tab", 49: "space", 50: "grv", 51: "backspace", 53: "esc",
	54: "rmet", 55: "lmet", 56: "lshift", 57: "caps", 58: "lalt", 59: "lctl",
	60: "rshift", 61: "ralt", 62: "rctl", 64: "f17", 65: "kp.", 67: "kp*",
	69: "kp+", 71: "clear", 75: "kp/", 76: "kpenter", 78: "kp-", 79: "f18",
	80: "f19", 82: "kp0", 83: "kp1", 84: "kp2", 85: "kp3", 86: "kp4",
	87: "kp5", 88: "kp6", 89: "kp7", 90: "f20", 91: "kp8", 92: "kp9",
	93: "yen", 94: "intlro", 96: "f5", 97: "f6", 98: "f7", 99: "f3",
	100: "f8", 101: "f9", 102: "muhenkan", 103: "f11", 104: "kana", 105: "f13",
	106: "f16", 107: "f14", 109: "f10", 111: "f12", 113: "f15",
	114: "help", 115: "home", 116: "pgup", 117: "del", 118: "f4",
	119: "end", 120: "f2", 121: "pgdn", 122: "f1", 123: "left",
	124: "right", 125: "down", 126: "up",
	mediaBase + 0: "volumeup", mediaBase + 1: "volumedown", mediaBase + 7: "volumemute",
	mediaBase + 16: "mediaplaypause", mediaBase + 17: "medianext", mediaBase + 18: "mediaprev",
	mediaBase + 19: "medianext", mediaBase + 20: "mediaprev",
}

type binding struct {
	code     int
	mapped   bool
	sequence []keys.Key
}

type remapper struct {
	mappings     *keyboard.Mappings
	destinations map[keys.Key]int
	active       map[int]binding
	held         map[int]int
	emit         func(code int, up, repeat bool)
}

func newRemapper(mappings *keyboard.Mappings, emit func(int, bool, bool)) *remapper {
	destinations := make(map[keys.Key]int)
	for code, name := range names {
		if key, ok := keys.Lookup(name); ok && code != mediaBase+19 && code != mediaBase+20 {
			destinations[key] = code
		}
	}
	return &remapper{mappings, destinations, make(map[int]binding), make(map[int]int), emit}
}

func (r *remapper) resolve(code int) binding {
	key, known := keys.Lookup(names[code])
	if !known {
		return binding{code: code}
	}
	mapping, ok := r.mappings.Resolve(key)
	if !ok {
		return binding{code: code}
	}
	if mapping.Noop {
		return binding{code: -1, mapped: true}
	}
	if len(mapping.SendKeys) != 0 {
		for _, key := range mapping.SendKeys {
			if _, ok := r.destinations[key]; !ok {
				return binding{code: code}
			}
		}
		return binding{code: -1, mapped: true, sequence: mapping.SendKeys}
	}
	if destination, ok := r.destinations[mapping.Destination]; ok && destination != code {
		return binding{code: destination, mapped: true}
	}
	return binding{code: code}
}

func (r *remapper) change(code int, up, repeat, mapped bool) bool {
	if repeat {
		if mapped {
			r.emit(code, false, true)
		}
		return mapped
	}
	before := r.held[code]
	if up {
		if before > 1 {
			r.held[code]--
			return true
		}
		delete(r.held, code)
	} else {
		r.held[code]++
		if before > 0 {
			return true
		}
	}
	if mapped {
		r.emit(code, up, false)
	}
	return mapped
}

func (r *remapper) Send(event keyboard.Event) error {
	r.change(r.destinations[event.Key], event.KeyUp, false, true)
	return nil
}

func (r *remapper) handle(code int, up, repeat bool) bool {
	b, active := r.active[code]
	switch {
	case up:
		if !active {
			return false
		}
		delete(r.active, code)
	case !active:
		if repeat {
			return false
		}
		b = r.resolve(code)
		r.active[code] = b
	case !repeat:
		return b.mapped
	}
	if b.code >= 0 {
		return r.change(b.code, up, repeat, b.mapped)
	}
	if !up {
		_ = keyboard.SendSequence(r, b.sequence)
	}
	return true
}

func (r *remapper) release() {
	for code, b := range r.active {
		if b.mapped && b.code >= 0 {
			r.change(b.code, true, false, true)
		}
		delete(r.active, code)
	}
}

const capsFlag = 1 << 16

var modifiers = []struct {
	code        int
	side, group uint64
}{
	{54, 0x10, 1 << 20},   // Right Command
	{55, 0x08, 1 << 20},   // Left Command
	{56, 0x02, 1 << 17},   // Left Shift
	{58, 0x20, 1 << 19},   // Left Option
	{59, 0x01, 1 << 18},   // Left Control
	{60, 0x04, 1 << 17},   // Right Shift
	{61, 0x40, 1 << 19},   // Right Option
	{62, 0x2000, 1 << 18}, // Right Control
}

func modifierSide(code int) uint64 {
	for _, mod := range modifiers {
		if mod.code == code {
			return mod.side
		}
	}
	return 0
}

func (r *remapper) outputFlags(flags uint64, suppressCaps, caps bool) uint64 {
	var physicalGroups uint64
	for _, mod := range modifiers {
		if flags&mod.side != 0 {
			physicalGroups |= mod.group
		}
	}
	flags &^= physicalGroups
	for _, mod := range modifiers {
		if b := r.active[mod.code]; b.mapped {
			flags &^= mod.side
		}
		if r.held[mod.code] != 0 {
			flags |= mod.side
		}
	}
	for _, mod := range modifiers {
		if flags&mod.side != 0 {
			flags |= mod.group
		}
	}
	if suppressCaps {
		flags &^= capsFlag
	}
	if caps {
		flags |= capsFlag
	}
	return flags
}

// mediaKey decodes the data1 payload of an NX_SUBTYPE_AUX_CONTROL_BUTTONS event.
func mediaKey(data uint32) (code int, up, repeat, ok bool) {
	const downState, upState = 0x0a, 0x0b
	state := (data >> 8) & 0xff
	code = mediaBase + int(data>>16)
	_, known := names[code]
	return code, state == upState, data&1 != 0, known && (state == downState || state == upState)
}
