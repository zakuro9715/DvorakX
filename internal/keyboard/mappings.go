package keyboard

import (
	"slices"
	"sync/atomic"

	"github.com/zakuro9715/DvorakX/internal/config"
)

// Mappings provides lock-free replacement of the active key map.
type Mappings struct {
	value atomic.Value
}

// Mapping describes the destination or action for a source key.
type Mapping struct {
	Destination uint16
	Noop        bool
	SendKeys    []uint16
}

// NewMappings creates mappings initialized with remap.
func NewMappings(remap map[uint16]uint16) *Mappings {
	mappings := new(Mappings)
	mappings.Replace(remap)
	return mappings
}

// NewMappingsWithActions creates mappings initialized with remaps and actions.
func NewMappingsWithActions(remap map[uint16]uint16, noop map[uint16]bool, sendKeys map[uint16][]uint16) *Mappings {
	mappings := new(Mappings)
	mappings.ReplaceWithActions(remap, noop, sendKeys)
	return mappings
}

// NewMappingsFromStages creates mappings that apply configuration stages in order.
func NewMappingsFromStages(stages []config.MappingStage) *Mappings {
	mappings := new(Mappings)
	mappings.ReplaceStages(stages)
	return mappings
}

// ReplaceStages atomically installs an ordered copy of configuration stages.
func (m *Mappings) ReplaceStages(stages []config.MappingStage) {
	copyStages := make([]config.MappingStage, len(stages))
	for index, stage := range stages {
		mappings := make(map[uint16]config.StageMapping, len(stage.Mappings))
		for source, mapping := range stage.Mappings {
			mapping.SendKeys = slices.Clone(mapping.SendKeys)
			mappings[source] = mapping
		}
		copyStages[index] = config.MappingStage{Mappings: mappings}
	}
	m.value.Store(copyStages)
}

// Load returns the current immutable key map.
func (m *Mappings) Load() map[uint16]uint16 {
	if stages, ok := m.value.Load().([]config.MappingStage); ok {
		remap := make(map[uint16]uint16)
		for source, mapping := range flattenStages(stages) {
			if !mapping.Noop && len(mapping.SendKeys) == 0 {
				remap[source] = mapping.Destination
			}
		}
		return remap
	}
	mappings := m.value.Load().(map[uint16]Mapping)
	remap := make(map[uint16]uint16, len(mappings))
	for source, mapping := range mappings {
		if !mapping.Noop && len(mapping.SendKeys) == 0 {
			remap[source] = mapping.Destination
		}
	}
	return remap
}

func flattenStages(stages []config.MappingStage) map[uint16]Mapping {
	keys := make(map[uint16]struct{})
	for _, stage := range stages {
		for source := range stage.Mappings {
			keys[source] = struct{}{}
		}
	}
	flattened := make(map[uint16]Mapping, len(keys))
	for source := range keys {
		if mapping, ok := resolveStages(source, stages); ok {
			flattened[source] = mapping
		}
	}
	return flattened
}

// Replace atomically installs a copy of remap.
func (m *Mappings) Replace(remap map[uint16]uint16) {
	m.ReplaceWithActions(remap, nil, nil)
}

// ReplaceWithActions atomically installs remaps and actions.
func (m *Mappings) ReplaceWithActions(remap map[uint16]uint16, noop map[uint16]bool, sendKeys map[uint16][]uint16) {
	copy := make(map[uint16]Mapping, len(remap)+len(noop)+len(sendKeys))
	for source, destination := range remap {
		copy[source] = Mapping{Destination: destination}
	}
	for source := range noop {
		copy[source] = Mapping{Noop: true}
	}
	for source, keys := range sendKeys {
		copy[source] = Mapping{SendKeys: slices.Clone(keys)}
	}
	m.value.Store(copy)
}

func (m *Mappings) lookup(key uint16) (Mapping, bool) {
	mappings, ok := m.value.Load().(map[uint16]Mapping)
	if !ok {
		return Mapping{}, false
	}
	mapping, ok := mappings[key]
	return mapping, ok
}

func (m *Mappings) lookupStages() ([]config.MappingStage, bool) {
	stages, ok := m.value.Load().([]config.MappingStage)
	if !ok {
		return nil, false
	}
	return stages, ok
}
