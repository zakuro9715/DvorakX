package parser

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func (p *validator) resolve(raw map[string]any) config.Config {
	loaded := config.Config{Layers: make(map[string]config.Layer)}
	if value, ok := raw["global"]; ok {
		if table := p.table(value, "global", "actions", "remaps"); table != nil {
			actions := p.parseActions(table["actions"], "global.actions")
			loaded.Global = p.remaps(table["remaps"], "global.remaps", actions)
		}
	}
	if len(raw) == 0 || (len(raw) == 1 && raw["global"] != nil) {
		if len(loaded.Global) == 0 && len(p.issues) == 0 {
			p.reportf("", "config must define at least one mapping")
		}
		return loaded
	}

	source := p.source(raw)
	for _, name := range slices.Sorted(maps.Keys(raw)) {
		switch name {
		case "global", "meta", "src":
			continue
		}
		path := joinPath("", name)
		if table := p.table(raw[name], path, "def", "actions", "remaps"); table != nil {
			actions := p.parseActions(table["actions"], path+".actions")
			loaded.Layers[name] = config.Layer{
				Definition: p.layerDefinition(table["def"], path+".def", actions, source),
				Remaps:     p.remaps(table["remaps"], path+".remaps", actions),
			}
		}
	}

	loaded.DefaultLayer = p.defaultLayer(raw["meta"], loaded.Layers)
	layer := loaded.Layers[loaded.DefaultLayer]
	if len(p.issues) == 0 && len(loaded.Global)+len(layer.Definition)+len(layer.Remaps) == 0 {
		p.reportValuef(joinPath("", loaded.DefaultLayer)+".def", "config must define at least one mapping")
	}
	return loaded
}

func (p *validator) defaultLayer(value any, layers map[string]config.Layer) string {
	if value == nil {
		p.reportf("meta", "layer configuration requires [meta]")
		return ""
	}
	meta := p.table(value, "meta", "default")
	if meta == nil {
		return ""
	}
	spelling, _ := meta["default"].(string)
	name, ok := strings.CutPrefix(spelling, "&")
	if !ok || name == "" {
		p.reportContentf("meta.default", "[meta].default must be a string layer reference with &")
		return ""
	}
	if _, exists := layers[name]; !exists {
		p.reportContentf("meta.default", "[meta].default references undefined layer %q", name)
		return ""
	}
	return name
}

func (p *validator) table(value any, path string, fields ...string) map[string]any {
	table, ok := value.(map[string]any)
	if !ok {
		p.reportValuef(path, "[%s] must be a TOML table", path)
		return nil
	}
	for _, field := range slices.Sorted(maps.Keys(table)) {
		if !slices.Contains(fields, field) {
			p.reportf(joinPath(path, field), "unsupported [%s] field %q", path, field)
		}
	}
	return table
}

func (p *validator) source(raw map[string]any) []keys.Key {
	if raw["src"] == nil {
		p.reportf("src", "layer configuration requires [src]")
		return nil
	}
	table := p.table(raw["src"], "src", "def")
	if table == nil {
		return nil
	}
	tokens := p.tokens(table["def"], "src.def")
	if tokens == nil {
		return nil
	}
	source := make([]keys.Key, len(tokens))
	seen := make(map[keys.Key]bool)
	valid := true
	for i, token := range tokens {
		path := indexPath("src.def", i)
		key, ok := keys.Lookup(token.value)
		switch {
		case !ok:
			p.reportf(path, "unsupported source layer key %q", token.value)
			valid = false
		case seen[key]:
			p.reportf(path, "duplicate source layer key %q", token.value)
			valid = false
		default:
			source[i] = key
			seen[key] = true
		}
	}
	if !valid {
		return nil
	}
	return source
}

func (p *validator) layerDefinition(value any, path string, actions map[string]config.Binding, source []keys.Key) config.Bindings {
	bindings := make(config.Bindings)
	if spelling, ok := value.(string); ok && strings.TrimSpace(spelling) == "#src" {
		for _, key := range source {
			bindings[key] = config.Binding{Kind: config.RemapKey, Destination: key}
		}
		return bindings
	}
	tokens := p.tokens(value, path)
	if tokens == nil {
		return nil
	}
	aligned := source != nil && len(source) == len(tokens)
	if source != nil && !aligned {
		p.reportValuef(path, "source and layer definitions have different lengths: %d and %d", len(source), len(tokens))
	}
	for i, token := range tokens {
		if !token.escaped && token.value == "_" {
			continue
		}
		binding, ok := p.binding(token, indexPath(path, i), "layer", actions)
		if ok && aligned {
			bindings[source[i]] = binding
		}
	}
	return bindings
}

func (p *validator) remaps(value any, path string, actions map[string]config.Binding) config.Bindings {
	bindings := make(config.Bindings)
	switch value := value.(type) {
	case nil:
	case map[string]any:
		p.mergeRemaps(bindings, value, path, actions)
	case []any:
		for i, entry := range value {
			entryPath := indexPath(path, i)
			if table, ok := entry.(map[string]any); ok {
				p.mergeRemaps(bindings, table, entryPath, actions)
			} else {
				p.reportf(entryPath, "%s must be a TOML table", entryPath)
			}
		}
	default:
		p.reportValuef(path, "[%s].remaps must be a TOML table or table array", strings.TrimSuffix(path, ".remaps"))
	}
	return bindings
}

func (p *validator) mergeRemaps(bindings config.Bindings, table map[string]any, path string, actions map[string]config.Binding) {
	seen := make(map[keys.Key]bool)
	for _, name := range slices.Sorted(maps.Keys(table)) {
		entry := joinPath(path, name)
		source, ok := p.literal(name, entry)
		if !ok {
			continue
		}
		key, ok := keys.Lookup(source.value)
		if !ok {
			p.reportf(entry, "unsupported remap source key %q", source.value)
			continue
		}
		if seen[key] {
			p.reportf(entry, "duplicate remap source key %q", name)
			continue
		}
		seen[key] = true
		destination, ok := table[name].(string)
		if !ok {
			p.reportValuef(entry, "invalid remap destination: expected string, got %s", tomlValueType(table[name]))
			continue
		}
		keyName, err := unescapeKeyName(destination)
		if err != nil {
			p.reportContentf(entry, "%v", err)
			continue
		}
		token := definitionToken{value: keyName, escaped: keyName != destination}
		if binding, ok := p.binding(token, entry, "remap destination", actions); ok {
			bindings[key] = binding
		}
	}
}

func tomlValueType(value any) string {
	switch value.(type) {
	case int64, float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "table"
	case time.Time, toml.LocalDateTime:
		return "date-time"
	case toml.LocalDate:
		return "date"
	case toml.LocalTime:
		return "time"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func (p *validator) binding(token definitionToken, path, context string, actions map[string]config.Binding) (config.Binding, bool) {
	if name, reference := strings.CutPrefix(token.value, "@"); reference && !token.escaped {
		binding, ok := actions[name]
		if !ok || name == "" {
			p.reportContentf(path, "undefined action %q", name)
			return config.Binding{}, false
		}
		return binding, true
	}
	key, ok := keys.Lookup(token.value)
	if !ok {
		p.reportContentf(path, "unsupported %s key %q", context, token.value)
		return config.Binding{}, false
	}
	return config.Binding{Kind: config.RemapKey, Destination: key}, true
}

func indexPath(path string, index int) string {
	return fmt.Sprintf("%s[%d]", path, index)
}
