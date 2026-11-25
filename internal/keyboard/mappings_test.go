package keyboard

import (
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestMappingsReplaceCopiesAndSwapsMap(t *testing.T) {
	initial := map[keys.Key]keys.Key{key(1): key(2)}
	mappings := NewMappings(initial)
	initial[key(1)] = key(3)

	if got := mappings.Load()[key(1)]; got != key(2) {
		t.Fatalf("initial mapping = %#v, want 2", got)
	}

	replacement := map[keys.Key]keys.Key{key(1): key(4)}
	mappings.Replace(replacement)
	replacement[key(1)] = key(5)

	if got := mappings.Load()[key(1)]; got != key(4) {
		t.Fatalf("replacement mapping = %#v, want 4", got)
	}
}

func key(scanCode uint16) keys.Key {
	return keys.Key{ScanCode: scanCode}
}
