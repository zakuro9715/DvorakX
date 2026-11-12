package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type fileConfig struct {
	Global globalConfig `toml:"global"`
}

type globalConfig struct {
	Remap map[string]string `toml:"remap"`
}

// Config is the supported subset of the DvorakX configuration.
type Config struct {
	Remap map[uint16]uint16
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

// Parse parses and validates the initial direct-remap TOML subset.
func Parse(data []byte) (Config, error) {
	var raw fileConfig
	decoder := toml.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if len(raw.Global.Remap) == 0 {
		return Config{}, fmt.Errorf("config must define at least one [global].remap entry")
	}

	remap := make(map[uint16]uint16, len(raw.Global.Remap))
	for sourceName, destinationName := range raw.Global.Remap {
		source, ok := virtualKey(sourceName)
		if !ok {
			return Config{}, fmt.Errorf("unsupported remap source key %q", sourceName)
		}
		destination, ok := virtualKey(destinationName)
		if !ok {
			return Config{}, fmt.Errorf("unsupported remap destination key %q", destinationName)
		}
		remap[source] = destination
	}
	return Config{Remap: remap}, nil
}
