package keyboard

import "testing"

func TestMappingsReplaceCopiesAndSwapsMap(t *testing.T) {
	initial := map[uint16]uint16{1: 2}
	mappings := NewMappings(initial)
	initial[1] = 3

	if got := mappings.Load()[1]; got != 2 {
		t.Fatalf("initial mapping = %d, want 2", got)
	}

	replacement := map[uint16]uint16{1: 4}
	mappings.Replace(replacement)
	replacement[1] = 5

	if got := mappings.Load()[1]; got != 4 {
		t.Fatalf("replacement mapping = %d, want 4", got)
	}
}
