package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
)

// Config contains the active static key remapping.
type Config struct {
	Remap map[uint16]uint16
}

type definitionToken struct {
	value   string
	escaped bool
}

// ResolveConfigPath returns the explicit path, or config.toml next to the executable.
func ResolveConfigPath(explicitPath, executablePath string) (string, error) {
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}
	if executablePath == "" {
		return "", fmt.Errorf("resolve default config path: executable path is empty")
	}
	return filepath.Join(filepath.Dir(executablePath), "config.toml"), nil
}

// LoadFile reads, parses, and validates a configuration file.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	return Parse(data)
}

// Parse parses and validates direct remaps and static layer mappings.
func Parse(data []byte) (Config, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	global, err := parseGlobal(raw)
	if err != nil {
		return Config{}, err
	}

	remap := make(map[uint16]uint16)
	hasLayers, err := resolveLayers(raw, remap)
	if err != nil {
		return Config{}, err
	}
	if !hasLayers && len(global) == 0 {
		return Config{}, fmt.Errorf("config must define at least one [global].remap entry")
	}
	for source, destination := range global {
		remap[source] = destination
	}
	if len(remap) == 0 {
		return Config{}, fmt.Errorf("config must define at least one mapping")
	}
	return Config{Remap: remap}, nil
}

func parseGlobal(raw map[string]any) (map[uint16]uint16, error) {
	value, ok := raw["global"]
	if !ok {
		return map[uint16]uint16{}, nil
	}
	table, err := asTable(value, "[global]")
	if err != nil {
		return nil, err
	}
	for name := range table {
		if name != "remap" {
			return nil, fmt.Errorf("unsupported [global] field %q", name)
		}
	}

	remapValue, ok := table["remap"]
	if !ok {
		return map[uint16]uint16{}, nil
	}
	remap := make(map[uint16]uint16)
	switch remapValue := remapValue.(type) {
	case map[string]any:
		if err := mergeRemap(remap, remapValue); err != nil {
			return nil, err
		}
	case []any:
		for index, value := range remapValue {
			entries, err := asTable(value, fmt.Sprintf("[global].remap[%d]", index))
			if err != nil {
				return nil, err
			}
			if err := mergeRemap(remap, entries); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("[global].remap must be a TOML table or table array")
	}
	return remap, nil
}

func mergeRemap(remap map[uint16]uint16, entries map[string]any) error {
	for sourceName, value := range entries {
		destinationName, ok := value.(string)
		if !ok {
			return fmt.Errorf("invalid remap destination for %q: must be a string", sourceName)
		}
		sourceName, err := unescapeKeyName(sourceName)
		if err != nil {
			return fmt.Errorf("invalid remap source key: %w", err)
		}
		destinationName, err = unescapeKeyName(destinationName)
		if err != nil {
			return fmt.Errorf("invalid remap destination key: %w", err)
		}
		source, ok := virtualKey(sourceName)
		if !ok {
			return fmt.Errorf("unsupported remap source key %q", sourceName)
		}
		destination, ok := virtualKey(destinationName)
		if !ok {
			return fmt.Errorf("unsupported remap destination key %q", destinationName)
		}
		remap[source] = destination
	}
	return nil
}

func resolveLayers(raw map[string]any, remap map[uint16]uint16) (bool, error) {
	_, hasMeta := raw["meta"]
	sourceValue, hasSource := raw["src"]
	var layerNames []string
	for name := range raw {
		switch name {
		case "global", "meta", "src":
		default:
			layerNames = append(layerNames, name)
		}
	}
	if !hasMeta && !hasSource {
		if len(layerNames) != 0 {
			return false, fmt.Errorf("layer table %q requires [meta] and [src]", layerNames[0])
		}
		return false, nil
	}
	if !hasMeta {
		return false, fmt.Errorf("layer configuration requires [meta]")
	}
	if !hasSource {
		return false, fmt.Errorf("layer configuration requires [src]")
	}

	meta, err := asTable(raw["meta"], "[meta]")
	if err != nil {
		return false, err
	}
	for name := range meta {
		if name != "default" {
			return false, fmt.Errorf("unsupported [meta] field %q", name)
		}
	}
	defaultValue, ok := meta["default"].(string)
	if !ok {
		return false, fmt.Errorf("[meta].default must be a string layer reference")
	}
	defaultLayer, ok := strings.CutPrefix(defaultValue, "@")
	if !ok || defaultLayer == "" {
		return false, fmt.Errorf("[meta].default must reference a layer with @")
	}

	source, err := parseDefinition(sourceValue, "[src]")
	if err != nil {
		return false, err
	}
	layerValue, ok := raw[defaultLayer]
	if !ok {
		return false, fmt.Errorf("[meta].default references undefined layer %q", defaultLayer)
	}
	for _, layerName := range layerNames {
		if _, err := parseDefinition(raw[layerName], "["+layerName+"]"); err != nil {
			return false, err
		}
	}
	destination, err := parseDefinition(layerValue, "["+defaultLayer+"]")
	if err != nil {
		return false, err
	}
	if len(destination) == 1 && !destination[0].escaped && destination[0].value == "#src" {
		destination = source
	}
	if len(source) != len(destination) {
		return false, fmt.Errorf("source and default layer definitions have different lengths: %d and %d", len(source), len(destination))
	}

	for index, sourceToken := range source {
		sourceKey, ok := virtualKey(sourceToken.value)
		if !ok {
			return false, fmt.Errorf("unsupported source layer key %q at position %d", sourceToken.value, index)
		}
		if !destination[index].escaped && destination[index].value == "_" {
			continue
		}
		destinationKey, ok := virtualKey(destination[index].value)
		if !ok {
			return false, fmt.Errorf("unsupported default layer key %q at position %d", destination[index].value, index)
		}
		remap[sourceKey] = destinationKey
	}
	return true, nil
}

func parseDefinition(value any, path string) ([]definitionToken, error) {
	table, err := asTable(value, path)
	if err != nil {
		return nil, err
	}
	for name := range table {
		if name != "def" {
			return nil, fmt.Errorf("unsupported %s field %q", path, name)
		}
	}
	definition, ok := table["def"]
	if !ok {
		return nil, fmt.Errorf("%s.def is required", path)
	}
	switch definition := definition.(type) {
	case string:
		if strings.TrimSpace(definition) == "#src" {
			return []definitionToken{{value: "#src"}}, nil
		}
		tokens, err := splitDefinition(definition)
		if err != nil {
			return nil, fmt.Errorf("%s.def: %w", path, err)
		}
		return tokens, nil
	case []any:
		keys := make([]definitionToken, len(definition))
		for index, value := range definition {
			key, ok := value.(string)
			if !ok {
				return nil, fmt.Errorf("%s.def entry at position %d must be a string", path, index)
			}
			unescaped, err := unescapeKeyName(key)
			if err != nil {
				return nil, fmt.Errorf("%s.def entry at position %d: %w", path, index, err)
			}
			keys[index] = definitionToken{value: unescaped, escaped: unescaped != key}
		}
		return keys, nil
	default:
		return nil, fmt.Errorf("%s.def must be a string or string array", path)
	}
}

func splitDefinition(definition string) ([]definitionToken, error) {
	var lines []string
	for _, line := range strings.Split(definition, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	var tokens []definitionToken
	var token strings.Builder
	escaped := false
	for _, character := range strings.Join(lines, " ") {
		switch {
		case escaped:
			switch character {
			case '{':
				return nil, fmt.Errorf("nested { is not allowed")
			case '}':
				if token.Len() == 0 {
					return nil, fmt.Errorf("{} is not allowed")
				}
				tokens = append(tokens, definitionToken{value: token.String(), escaped: true})
				token.Reset()
				escaped = false
			default:
				token.WriteRune(character)
			}
		case character == '{':
			if token.Len() != 0 {
				return nil, fmt.Errorf("{ must start a key definition")
			}
			escaped = true
		case character == '}':
			return nil, fmt.Errorf("unexpected }")
		case unicode.IsSpace(character) || character == ',' || character == '|':
			if token.Len() != 0 {
				tokens = append(tokens, definitionToken{value: token.String()})
				token.Reset()
			}
		default:
			token.WriteRune(character)
		}
	}
	if escaped {
		return nil, fmt.Errorf("unclosed {")
	}
	if token.Len() != 0 {
		tokens = append(tokens, definitionToken{value: token.String()})
	}
	return tokens, nil
}

func asTable(value any, path string) (map[string]any, error) {
	table, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s must be a TOML table", path)
	}
	return table, nil
}

func unescapeKeyName(name string) (string, error) {
	if !strings.ContainsAny(name, "{}") {
		return name, nil
	}
	if len(name) < 3 || name[0] != '{' || name[len(name)-1] != '}' {
		return "", fmt.Errorf("%q must be enclosed by one pair of braces", name)
	}
	unescaped := name[1 : len(name)-1]
	if unescaped == "" || strings.ContainsAny(unescaped, "{}") {
		return "", fmt.Errorf("%q contains invalid braces", name)
	}
	return unescaped, nil
}
