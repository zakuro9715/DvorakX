package keyboard

import (
	"fmt"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

// Mappings provides lock-free replacement of the active key map.
type Mappings struct {
	value atomic.Value
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
func NewMappingsFromConfig(loaded config.Config) (*Mappings, error) {
	stages, err := stagesFromConfig(loaded)
	if err != nil {
		return nil, err
	}
	mappings := new(Mappings)
	mappings.ReplaceStages(stages)
	return mappings, nil
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
	m.value.Store(copyStages)
}

// ReplaceConfig atomically installs mappings built from a configuration.
func (m *Mappings) ReplaceConfig(loaded config.Config) error {
	stages, err := stagesFromConfig(loaded)
	if err != nil {
		return err
	}
	m.ReplaceStages(stages)
	return nil
}

// Load returns the current immutable key map.
func (m *Mappings) Load() map[keys.Key]keys.Key {
	if stages, ok := m.value.Load().([]MappingStage); ok {
		remap := make(map[keys.Key]keys.Key)
		for source, mapping := range flattenStages(stages) {
			if !mapping.Noop && len(mapping.SendKeys) == 0 {
				remap[source] = mapping.Destination
			}
		}
		return remap
	}
	mappings := m.value.Load().(map[keys.Key]Mapping)
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
	m.value.Store(copy)
}

func (m *Mappings) lookup(key keys.Key) (Mapping, bool) {
	mappings, ok := m.value.Load().(map[keys.Key]Mapping)
	if !ok {
		return Mapping{}, false
	}
	mapping, ok := mappings[key]
	return mapping, ok
}

func (m *Mappings) lookupStages() ([]MappingStage, bool) {
	stages, ok := m.value.Load().([]MappingStage)
	if !ok {
		return nil, false
	}
	return stages, ok
}

func stagesFromConfig(loaded config.Config) ([]MappingStage, error) {
	stages := make([]MappingStage, 1, 3)
	stages[0] = remapStage(loaded.Global)
	defaultLayer, ok := strings.CutPrefix(loaded.Meta.Default, "&")
	if !ok || defaultLayer == "" {
		return stages, nil
	}
	layer, ok := loaded.Layers[defaultLayer]
	if !ok {
		return nil, fmt.Errorf("default layer %q is undefined", defaultLayer)
	}
	source := loaded.Source.Definition.Tokens
	destination := layer.Definition.Tokens
	if len(destination) == 1 && destination[0] == "#src" {
		destination = source
	}
	if len(source) != len(destination) {
		return nil, fmt.Errorf("source and default layer definitions have different lengths: %d and %d", len(source), len(destination))
	}
	definition := MappingStage{Mappings: make(map[keys.Key]StageMapping, len(source))}
	for index, sourceName := range source {
		sourceName, err := config.UnescapeKeyName(sourceName)
		if err != nil {
			return nil, fmt.Errorf("invalid source layer key %q: %w", sourceName, err)
		}
		sourceKey, ok := keys.Lookup(sourceName)
		if !ok {
			return nil, fmt.Errorf("unsupported source layer key %q", sourceName)
		}
		target := destination[index]
		target, err = config.UnescapeKeyName(target)
		if err != nil {
			return nil, fmt.Errorf("invalid layer key %q: %w", target, err)
		}
		if action, ok := actionReference(target, layer.Actions); ok {
			definition.Mappings[sourceKey] = actionMapping(action)
			continue
		}
		if target == "_" {
			definition.Mappings[sourceKey] = StageMapping{Passthrough: true}
			continue
		}
		destinationKey, ok := keys.Lookup(target)
		if !ok {
			return nil, fmt.Errorf("unsupported default layer key %q", target)
		}
		definition.Mappings[sourceKey] = StageMapping{Destination: destinationKey}
	}
	stages = append(stages, definition, remapStage(layer))
	return stages, nil
}

func remapStage(layer config.LayerConfig) MappingStage {
	stage := MappingStage{Mappings: make(map[keys.Key]StageMapping)}
	for _, entries := range layer.Remaps {
		for sourceName, destinationName := range entries {
			sourceName, sourceErr := config.UnescapeKeyName(sourceName)
			destinationName, destinationErr := config.UnescapeKeyName(destinationName)
			if sourceErr != nil || destinationErr != nil {
				continue
			}
			source, ok := keys.Lookup(sourceName)
			if !ok {
				continue
			}
			if action, ok := actionReference(destinationName, layer.Actions); ok {
				stage.Mappings[source] = actionMapping(action)
				continue
			}
			if destination, ok := keys.Lookup(destinationName); ok {
				stage.Mappings[source] = StageMapping{Destination: destination}
			}
		}
	}
	return stage
}

func actionReference(value string, actions map[string]config.ActionSpec) (config.ActionSpec, bool) {
	name, ok := strings.CutPrefix(value, "@")
	if !ok {
		return config.ActionSpec{}, false
	}
	action, ok := actions[name]
	return action, ok
}

func actionMapping(action config.ActionSpec) StageMapping {
	switch action.Kind {
	case config.NoopAction:
		return StageMapping{Noop: true}
	case config.SendKeysAction:
		return StageMapping{SendKeys: slices.Clone(action.Keys)}
	default:
		return StageMapping{Passthrough: true}
	}
}
