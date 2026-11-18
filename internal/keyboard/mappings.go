package keyboard

import "sync/atomic"

// Mappings provides lock-free replacement of the active key map.
type Mappings struct {
	value atomic.Value
}

// Mapping describes the destination or action for a source key.
type Mapping struct {
	Destination uint16
	Noop        bool
}

// NewMappings creates mappings initialized with remap.
func NewMappings(remap map[uint16]uint16) *Mappings {
	mappings := new(Mappings)
	mappings.Replace(remap)
	return mappings
}

// NewMappingsWithActions creates mappings initialized with remaps and actions.
func NewMappingsWithActions(remap map[uint16]uint16, noop map[uint16]bool) *Mappings {
	mappings := new(Mappings)
	mappings.ReplaceWithActions(remap, noop)
	return mappings
}

// Load returns the current immutable key map.
func (m *Mappings) Load() map[uint16]uint16 {
	mappings := m.value.Load().(map[uint16]Mapping)
	remap := make(map[uint16]uint16, len(mappings))
	for source, mapping := range mappings {
		if !mapping.Noop {
			remap[source] = mapping.Destination
		}
	}
	return remap
}

// Replace atomically installs a copy of remap.
func (m *Mappings) Replace(remap map[uint16]uint16) {
	m.ReplaceWithActions(remap, nil)
}

// ReplaceWithActions atomically installs remaps and actions.
func (m *Mappings) ReplaceWithActions(remap map[uint16]uint16, noop map[uint16]bool) {
	copy := make(map[uint16]Mapping, len(remap)+len(noop))
	for source, destination := range remap {
		copy[source] = Mapping{Destination: destination}
	}
	for source := range noop {
		copy[source] = Mapping{Noop: true}
	}
	m.value.Store(copy)
}

func (m *Mappings) lookup(key uint16) (Mapping, bool) {
	mappings := m.value.Load().(map[uint16]Mapping)
	mapping, ok := mappings[key]
	return mapping, ok
}
