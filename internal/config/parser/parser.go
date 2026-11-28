package parser

import (
	"fmt"
	"os"
	"sort"

	"github.com/pelletier/go-toml/v2"

	"github.com/zakuro9715/DvorakZ/internal/config"
)

// LoadFile reads, parses, and validates a configuration file.
func LoadFile(path string) (config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config.Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	return parse(data, path)
}

// Parse decodes TOML and resolves all configured bindings.
func Parse(data []byte) (config.Config, error) {
	return parse(data, "")
}

func parse(data []byte, path string) (config.Config, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return config.Config{}, syntaxError(data, path, err)
	}
	positions, err := collectSourcePositions(data)
	if err != nil {
		return config.Config{}, fmt.Errorf("index config: %w", err)
	}
	p := validator{positions: positions, path: path, lines: splitSourceLines(data)}
	loaded := p.resolve(raw)
	if len(p.issues) != 0 {
		sort.SliceStable(p.issues, func(i, j int) bool {
			left, right := p.issues[i].at, p.issues[j].at
			if left.Line != right.Line {
				return left.Line < right.Line
			}
			return left.Column < right.Column
		})
		issues := make([]error, len(p.issues))
		for i, issue := range p.issues {
			issues[i] = issue
		}
		return config.Config{}, &ValidationError{issues: issues}
	}
	return loaded, nil
}

type validator struct {
	positions sourcePositions
	path      string
	lines     []string
	issues    []*FieldError
}
