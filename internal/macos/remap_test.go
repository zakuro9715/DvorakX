//go:build darwin

package macos

import (
	"reflect"
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func key(t *testing.T, name string) keys.Key {
	t.Helper()
	result, ok := keys.Lookup(name)
	if !ok {
		t.Fatalf("unknown test key %q", name)
	}
	return result
}

func TestNativeKeyNames(t *testing.T) {
	r := newRemapper(nil, nil)
	for code, name := range names {
		key(t, name)
		if code == mediaBase+19 || code == mediaBase+20 {
			continue
		}
		if got := r.destinations[key(t, name)]; got != code {
			t.Errorf("%s maps to %d, want %d", name, got, code)
		}
	}
}

func TestRemapper(t *testing.T) {
	type event struct {
		code       int
		up, repeat bool
	}
	var got []event
	a, b, s := key(t, "a"), key(t, "b"), key(t, "s")
	volume := key(t, "volumeup")
	mappings := keyboard.NewMappings(map[keys.Key]keys.Key{a: b, s: b, volume: a})
	r := newRemapper(mappings, func(code int, up, repeat bool) {
		got = append(got, event{code, up, repeat})
	})

	// Hold two sources for the same output and reload before releasing them.
	r.handle(0, false, false) // A
	r.handle(1, false, false) // S
	r.handle(1, false, true)  // S repeat
	mappings.Replace(map[keys.Key]keys.Key{a: volume})
	r.handle(0, true, false) // A
	r.handle(1, true, false) // S

	// A fresh press uses the new volume mapping. Shutdown releases it.
	r.handle(0, false, false) // A
	r.release()
	want := []event{{11, false, false}, {11, false, true}, {11, true, false}, {mediaBase, false, false}, {mediaBase, true, false}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
}

func TestActions(t *testing.T) {
	mappings := keyboard.NewMappingsWithActions(nil,
		map[keys.Key]bool{key(t, "a"): true},
		map[keys.Key][]keys.Key{
			key(t, "s"): {key(t, "lmet"), key(t, "c")},
			key(t, "d"): {key(t, "c"), key(t, "f24")},
		})
	var got []int
	r := newRemapper(mappings, func(code int, up, _ bool) {
		if up {
			code = -code
		}
		got = append(got, code)
	})

	// Consume a disabled key and emit a Command+C chord for S.
	if !r.handle(0, false, false) || !r.handle(0, true, false) ||
		!r.handle(1, false, false) || !r.handle(1, true, false) {
		t.Fatal("expected configured actions to consume the source")
	}
	// Pass through D and emit nothing for its unsupported sequence.
	if r.handle(2, false, false) || r.handle(2, true, false) {
		t.Fatal("expected unsupported sequence to pass through")
	}
	if want := []int{55, 8, -8, -55}; !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func TestModifierFlags(t *testing.T) {
	mappings := keyboard.NewMappings(map[keys.Key]keys.Key{key(t, "lctl"): key(t, "rmet")})
	r := newRemapper(mappings, func(int, bool, bool) {})
	const ctrl, command = 1 << 18, 1 << 20

	// Hold both Controls with only Left Control mapped to Right Command.
	r.handle(59, false, false) // Left Control
	r.handle(62, false, false) // Right Control
	input := ctrl | modifierSide(59) | modifierSide(62)
	want := ctrl | command | modifierSide(62) | modifierSide(54)
	if got := r.outputFlags(input, false, false); got != want {
		t.Fatalf("flags = %#x, want %#x", got, want)
	}

	// Releasing Left Control also releases the synthetic Right Command.
	r.handle(59, true, false) // Left Control
	want = ctrl | modifierSide(62)
	if got := r.outputFlags(want, false, false); got != want {
		t.Fatalf("flags after release = %#x, want %#x", got, want)
	}
}

func TestMediaKey(t *testing.T) {
	// Decode volume up (native code zero), media repeat, and track release.
	for _, test := range []struct {
		data           uint32
		code           int
		up, repeat, ok bool
	}{
		{0x00000a00, mediaBase, false, false, true},      // Volume Up
		{0x00100a01, mediaBase + 16, false, true, true},  // Play/Pause
		{0x00140b00, mediaBase + 20, true, false, true},  // Rewind
		{0x00020a00, mediaBase + 2, false, false, false}, // Brightness Up
		{0x00000000, mediaBase, false, false, false},     // Invalid transition
	} {
		code, up, repeat, ok := mediaKey(test.data)
		if code != test.code || up != test.up || repeat != test.repeat || ok != test.ok {
			t.Errorf("payload %#x decoded to (%d, %v, %v, %v)", test.data, code, up, repeat, ok)
		}
	}
}
