//go:build amd64 || arm64

package linux

import (
	"fmt"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

type binding struct {
	code     uint16
	sequence []keys.Key
}

type remapper struct {
	mappings *keyboard.Mappings
	write    func(uint16, int32) error
	active   map[uint16]binding
	held     map[uint16]int
}

func newRemapper(mappings *keyboard.Mappings, write func(uint16, int32) error) *remapper {
	return &remapper{mappings: mappings, write: write, active: make(map[uint16]binding), held: make(map[uint16]int)}
}

func (r *remapper) change(code uint16, down bool) error {
	if down {
		r.held[code]++
		if r.held[code] == 1 {
			return r.write(code, 1)
		}
		return nil
	}
	switch r.held[code] {
	case 0:
		return nil
	case 1:
		if err := r.write(code, 0); err != nil {
			return err
		}
		delete(r.held, code)
	default:
		r.held[code]--
	}
	return nil
}

func (r *remapper) Send(event keyboard.Event) error {
	code, ok := keyToCode[event.Key]
	if !ok {
		return fmt.Errorf("unsupported Linux key %s", keys.Name(event.Key))
	}
	return r.change(code, !event.KeyUp)
}

func (r *remapper) resolve(code uint16) binding {
	key, known := codeToKey[code]
	if !known {
		return binding{code: code}
	}
	mapping, ok := r.mappings.Resolve(key)
	if !ok {
		return binding{code: code}
	}
	if mapping.Noop || len(mapping.SendKeys) > 0 {
		return binding{sequence: mapping.SendKeys}
	}
	if destination, ok := keyToCode[mapping.Destination]; ok {
		return binding{code: destination}
	}
	return binding{code: code}
}

func (r *remapper) handle(code uint16, value int32, recovery bool) error {
	if code == 0 || code > keyMax {
		return nil
	}
	b, active := r.active[code]
	switch value {
	case 0:
		delete(r.active, code)
		if b.code != 0 {
			return r.change(b.code, false)
		}
	case 1:
		if active {
			return nil
		}
		b = r.resolve(code)
		r.active[code] = b
		if b.code != 0 {
			return r.change(b.code, true)
		}
		if !recovery {
			return keyboard.SendSequence(r, b.sequence)
		}
	case 2:
		if b.code != 0 {
			return r.write(b.code, 2)
		}
		return keyboard.SendSequence(r, b.sequence)
	}
	return nil
}

func (r *remapper) sync(bits keyBits) error {
	for code := range r.active {
		if !bits.has(code) {
			if err := r.handle(code, 0, true); err != nil {
				return err
			}
		}
	}
	for code := uint16(1); code <= keyMax; code++ {
		if bits.has(code) {
			if err := r.handle(code, 1, true); err != nil {
				return err
			}
		}
	}
	return nil
}
