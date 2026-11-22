package keys

import "testing"

func TestLookup(t *testing.T) {
	tests := []struct {
		name string
		want uint16
	}{
		{name: "A", want: VKA},
		{name: "ctrl", want: VKControl},
		{name: "ENTER", want: VKReturn},
		{name: "{", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := Lookup(test.name)
			if test.want == 0 {
				if ok {
					t.Fatalf("Lookup(%q) = %#x, want unknown", test.name, got)
				}
				return
			}
			if !ok || got != test.want {
				t.Fatalf("Lookup(%q) = (%#x, %t), want (%#x, true)", test.name, got, ok, test.want)
			}
		})
	}
}
