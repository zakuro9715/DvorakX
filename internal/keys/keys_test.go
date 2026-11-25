package keys

import "testing"

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

func TestNameUnknownKey(t *testing.T) {
	if got := Name(Key{ScanCode: 0x99}); got != "0x99" {
		t.Errorf("Name(0x99) = %q, want %q", got, "0x99")
	}
}
