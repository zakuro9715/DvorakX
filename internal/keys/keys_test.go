package keys

import (
	"testing"
)

func TestLookup(t *testing.T) {
	tests := []struct {
		name string
		want Key
	}{
		{name: "A", want: Key{ScanCode: 0x1E}},
		{name: "ctrl", want: Key{ScanCode: 0x1D}},
		{name: "rctl", want: Key{ScanCode: 0x1D, Extended: true}},
		{name: "ENTER", want: Key{ScanCode: 0x1C}},
		{name: "{", want: Key{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := Lookup(test.name)
			if test.want == (Key{}) {
				if ok {
					t.Fatalf("Lookup(%q) = %#v, want unknown", test.name, got)
				}
				return
			}
			if !ok || got != test.want {
				t.Fatalf("Lookup(%q) = (%#v, %t), want (%#v, true)", test.name, got, ok, test.want)
			}
		})
	}
}

func TestLookupNamedKeys(t *testing.T) {
	tests := []struct {
		name string
		want Key
	}{
		{name: "F1", want: Key{ScanCode: F1}},
		{name: "F12", want: Key{ScanCode: F12}},
		{name: "F24", want: Key{ScanCode: F24}},
		{name: "home", want: Key{ScanCode: Home, Extended: true}},
		{name: "PAGEUP", want: Key{ScanCode: PageUp, Extended: true}},
		{name: "printscreen", want: Key{ScanCode: PrintScreen, Extended: true}},
		{name: "pagedown", want: Key{ScanCode: PageDown, Extended: true}},
		{name: "delete", want: Key{ScanCode: Delete, Extended: true}},
		{name: "numpad5", want: Key{ScanCode: Numpad5}},
		{name: "kp*", want: Key{ScanCode: NumpadMultiply}},
		{name: "numpad/", want: Key{ScanCode: Slash, Extended: true}},
		{name: "hiragana", want: Key{ScanCode: Kana}},
		{name: "convert", want: Key{ScanCode: Henkan}},
		{name: "nonconvert", want: Key{ScanCode: Muhenkan}},
		{name: "lang1", want: Key{ScanCode: Hangul}},
		{name: "lang2", want: Key{ScanCode: Hanja}},
		{name: "intlyen", want: Key{ScanCode: Yen}},
		{name: "iso", want: Key{ScanCode: IntlBackslash}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := Lookup(test.name)
			if !ok || got != test.want {
				t.Fatalf("Lookup(%q) = (%#v, %t), want (%#v, true)", test.name, got, ok, test.want)
			}
		})
	}
}

func TestNameNamedKeys(t *testing.T) {
	tests := []struct {
		key  Key
		want string
	}{
		{key: Key{ScanCode: F1}, want: "f1"},
		{key: Key{ScanCode: F24}, want: "f24"},
		{key: Key{ScanCode: NumLock}, want: "numlock"},
		{key: Key{ScanCode: NumpadMultiply}, want: "kp*"},
		{key: Key{ScanCode: Home, Extended: true}, want: "home"},
		{key: Key{ScanCode: PageDown, Extended: true}, want: "pgdn"},
		{key: Key{ScanCode: Delete, Extended: true}, want: "del"},
		{key: Key{ScanCode: Enter, Extended: true}, want: "kpenter"},
	}
	for _, test := range tests {
		if got := Name(test.key); got != test.want {
			t.Errorf("Name(%#v) = %q, want %q", test.key, got, test.want)
		}
	}
}

func TestNameUnknownKey(t *testing.T) {
	if got := Name(Key{ScanCode: 0x99}); got != "0x99" {
		t.Errorf("Name(0x99) = %q, want %q", got, "0x99")
	}
}

func TestCanonicalsNameAndLookupRoundTrip(t *testing.T) {
	for wantKey, name := range canonical {
		t.Run(name, func(t *testing.T) {
			gotKey, ok := Lookup(name)
			if !ok {
				t.Fatalf("Lookup(%q) failed", name)
			}
			if gotKey != wantKey {
				t.Fatalf("Lookup(%q) = %#v, want %#v", name, gotKey, wantKey)
			}
			if gotName := Name(gotKey); gotName != name {
				t.Errorf("Name(%#v) = %q, want %q", gotKey, gotName, name)
			}
		})
	}
}
