package keyboard

import (
	"slices"
	"sync/atomic"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

// Mappings provides lock-free replacement of the active key map.
type Mappings struct {
	value atomic.Value
}

type mappingSnapshot struct {
	direct map[keys.Key]Mapping
	stages []MappingStage
}

// MappingStage is one ordered step in keyboard processing.
type MappingStage struct {
	Mappings map[keys.Key]StageMapping
}

// StageMapping describes a transformation in one mapping stage.
type StageMapping struct {
	Destination keys.Key
	Noop        bool
	SendKeys    []keys.Key
	Passthrough bool
}

// Mapping describes the destination or action for a source key.
type Mapping struct {
	Destination keys.Key
	Noop        bool
	SendKeys    []keys.Key
}

// NewMappings creates mappings initialized with remap.
func NewMappings(remap map[keys.Key]keys.Key) *Mappings {
	mappings := new(Mappings)
	mappings.Replace(remap)
	return mappings
}

// NewMappingsWithActions creates mappings initialized with remaps and actions.
func NewMappingsWithActions(remap map[keys.Key]keys.Key, noop map[keys.Key]bool, sendKeys map[keys.Key][]keys.Key) *Mappings {
	mappings := new(Mappings)
	mappings.ReplaceWithActions(remap, noop, sendKeys)
	return mappings
}

// NewMappingsFromConfig creates mappings that apply the configured definitions in order.
func NewMappingsFromConfig(loaded config.Config) *Mappings {
	mappings := new(Mappings)
	mappings.ReplaceConfig(loaded)
	return mappings
}

// ReplaceStages atomically installs an ordered copy of configuration stages.
func (m *Mappings) ReplaceStages(stages []MappingStage) {
	copyStages := make([]MappingStage, len(stages))
	for index, stage := range stages {
		mappings := make(map[keys.Key]StageMapping, len(stage.Mappings))
		for source, mapping := range stage.Mappings {
			mapping.SendKeys = slices.Clone(mapping.SendKeys)
			mappings[source] = mapping
		}
		copyStages[index] = MappingStage{Mappings: mappings}
	}
	m.value.Store(mappingSnapshot{stages: copyStages})
}

// ReplaceConfig atomically installs mappings built from a configuration.
func (m *Mappings) ReplaceConfig(loaded config.Config) {
	stages := []MappingStage{bindingStage(loaded.Global)}
	if layer, ok := loaded.Layers[loaded.DefaultLayer]; ok {
		stages = append(stages, bindingStage(layer.Definition), bindingStage(layer.Remaps))
	}
	m.ReplaceStages(stages)
}

// Load returns the current immutable key map.
func (m *Mappings) Load() map[keys.Key]keys.Key {
	snapshot, _ := m.value.Load().(mappingSnapshot)
	mappings := snapshot.direct
	if snapshot.stages != nil {
		mappings = flattenStages(snapshot.stages)
	}
	remap := make(map[keys.Key]keys.Key, len(mappings))
	for source, mapping := range mappings {
		if !mapping.Noop && len(mapping.SendKeys) == 0 {
			remap[source] = mapping.Destination
		}
	}
	return remap
}

func flattenStages(stages []MappingStage) map[keys.Key]Mapping {
	keySet := make(map[keys.Key]struct{})
	for _, stage := range stages {
		for source := range stage.Mappings {
			keySet[source] = struct{}{}
		}
	}
	flattened := make(map[keys.Key]Mapping, len(keySet))
	for source := range keySet {
		if mapping, ok := resolveStages(source, stages); ok {
			flattened[source] = mapping
		}
	}
	return flattened
}

// Replace atomically installs a copy of remap.
func (m *Mappings) Replace(remap map[keys.Key]keys.Key) {
	m.ReplaceWithActions(remap, nil, nil)
}

// ReplaceWithActions atomically installs remaps and actions.
func (m *Mappings) ReplaceWithActions(remap map[keys.Key]keys.Key, noop map[keys.Key]bool, sendKeys map[keys.Key][]keys.Key) {
	copy := make(map[keys.Key]Mapping, len(remap)+len(noop)+len(sendKeys))
	for source, destination := range remap {
		copy[source] = Mapping{Destination: destination}
	}
	for source := range noop {
		copy[source] = Mapping{Noop: true}
	}
	for source, keys := range sendKeys {
		copy[source] = Mapping{SendKeys: slices.Clone(keys)}
	}
	m.value.Store(mappingSnapshot{direct: copy})
}

func bindingStage(bindings config.Bindings) MappingStage {
	stage := MappingStage{Mappings: make(map[keys.Key]StageMapping, len(bindings))}
	for key, binding := range bindings {
		switch binding.Kind {
		case config.RemapKey:
			stage.Mappings[key] = StageMapping{Destination: binding.Destination}
		case config.NoopAction:
			stage.Mappings[key] = StageMapping{Noop: true}
		case config.SendKeysAction:
			stage.Mappings[key] = StageMapping{SendKeys: binding.Keys}
		}
	}
	return stage
}
