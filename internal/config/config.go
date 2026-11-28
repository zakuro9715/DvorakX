package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

// Config contains the validated configuration file structure.
type Config struct {
	Meta   MetaConfig
	Global LayerConfig
	Source LayerConfig
	Layers map[string]LayerConfig
}

// MetaConfig contains settings from the [meta] table.
type MetaConfig struct {
	Default string
}

// LayerConfig contains a layer definition, actions, and direct remaps.
type LayerConfig struct {
	Definition Definition
	Actions    map[string]ActionSpec
	Remaps     []map[string]string
}

// Definition preserves the source form of a layer definition.
type Definition struct {
	Text   string
	Keys   []string
	Tokens []string
}

// Action identifies a configured action.
type Action uint8

const (
	_ Action = iota
	NoopAction
	SendKeysAction
)

// ActionSpec contains an action and its resolved key arguments.
type ActionSpec struct {
	Kind Action
	Keys []keys.Key
}

// ResolveConfigPath returns the explicit path, or config.toml next to the executable.
func ResolveConfigPath(explicitPath, executablePath string) (string, error) {
	if explicitPath != "" {
		return filepath.Clean(explicitPath), nil
	}
	if executablePath == "" {
		return "", errors.New("resolve default config path: executable path is empty")
	}
	return filepath.Join(filepath.Dir(executablePath), "config.toml"), nil
}

// UnescapeKeyName removes the explicit braces used for punctuation key names.
func UnescapeKeyName(name string) (string, error) {
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
