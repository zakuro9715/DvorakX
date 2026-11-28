package parser

import (
	"maps"
	"slices"
	"strings"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func (p *validator) parseActions(value any, path string) map[string]config.Binding {
	actions := make(map[string]config.Binding)
	if value == nil {
		return actions
	}
	table, ok := value.(map[string]any)
	if !ok {
		p.reportValuef(path, "%s must be a TOML table", path)
		return actions
	}
	for _, name := range slices.Sorted(maps.Keys(table)) {
		body, ok := table[name].(map[string]any)
		if !ok || len(body) != 1 {
			p.reportValuef(joinPath(path, name), "action %q must contain exactly one action", name)
			continue
		}
		switch {
		case body["noop"] != nil:
			arguments, ok := body["noop"].([]any)
			if !ok || len(arguments) != 0 {
				p.reportValuef(joinPath(path, name)+".noop", "noop action does not accept arguments")
				continue
			}
			actions[name] = config.Binding{Kind: config.NoopAction}
		case body["send-keys"] != nil:
			keys, ok := p.parseSendKeys(body["send-keys"], joinPath(path, name)+".send-keys")
			if !ok {
				continue
			}
			actions[name] = config.Binding{Kind: config.SendKeysAction, Keys: keys}
		default:
			for kind := range body {
				p.reportf(joinPath(joinPath(path, name), kind), "unsupported action in %q", name)
			}
		}
	}
	return actions
}

func (p *validator) parseSendKeys(value any, path string) ([]keys.Key, bool) {
	var names []string
	switch value := value.(type) {
	case string:
		names = strings.Fields(value)
	case []any:
		names = make([]string, len(value))
		for index, item := range value {
			name, ok := item.(string)
			if !ok {
				p.reportf(indexPath(path, index), "send-keys argument at position %d must be a string", index)
				return nil, false
			}
			names[index] = name
		}
	default:
		p.reportValuef(path, "send-keys must be a string or string array")
		return nil, false
	}
	if len(names) == 0 {
		p.reportValuef(path, "send-keys must contain at least one key")
		return nil, false
	}
	resolved := make([]keys.Key, len(names))
	for index, name := range names {
		key, ok := keys.Lookup(name)
		if !ok {
			p.reportf(indexPath(path, index), "unsupported send-keys key %q", name)
			return nil, false
		}
		resolved[index] = key
	}
	return resolved, true
}
