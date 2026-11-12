package keyboard

import "sync/atomic"

// Mappings provides lock-free replacement of the active key map.
type Mappings struct {
	value atomic.Value
}

// NewMappings creates mappings initialized with remap.
func NewMappings(remap map[uint16]uint16) *Mappings {
	mappings := new(Mappings)
	mappings.Replace(remap)
	return mappings
}

// Load returns the current immutable key map.
func (m *Mappings) Load() map[uint16]uint16 {
	return m.value.Load().(map[uint16]uint16)
}

// Replace atomically installs a copy of remap.
func (m *Mappings) Replace(remap map[uint16]uint16) {
	copy := make(map[uint16]uint16, len(remap))
	for source, destination := range remap {
		copy[source] = destination
	}
	m.value.Store(copy)
}
