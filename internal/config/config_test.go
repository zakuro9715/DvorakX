package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemap(t *testing.T) {
	config, err := Parse([]byte("[global]\nremap = { caps = 'ctrl', a = 'b' }\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(config.Remap) != 2 {
		t.Fatalf("len(Remap) = %d, want 2", len(config.Remap))
	}
	if config.Remap[vkCapital] != vkControl {
		t.Errorf("caps remap = %#x, want %#x", config.Remap[vkCapital], vkControl)
	}
	if config.Remap[vkA] != vkA+1 {
		t.Errorf("a remap = %#x, want %#x", config.Remap[vkA], vkA+1)
	}
}

func TestParseRejectsUnsupportedConfiguration(t *testing.T) {
	_, err := Parse([]byte("[global]\norder = ['qwerty']\n"))
	if err == nil || !strings.Contains(err.Error(), "strict mode") {
		t.Fatalf("Parse() error = %v, want strict-mode error", err)
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse([]byte("[global]\nremap = { capslock = 'ctrl' }\n"))
	if err == nil || !strings.Contains(err.Error(), `unsupported remap source key "capslock"`) {
		t.Fatalf("Parse() error = %v, want unsupported key error", err)
	}
}

func TestParseRejectsInvalidRemapValue(t *testing.T) {
	_, err := Parse([]byte("[global]\nremap = { caps = 1 }\n"))
	if err == nil || !strings.Contains(err.Error(), "cannot decode") {
		t.Fatalf("Parse() error = %v, want TOML type error", err)
	}
}

func TestResolveConfigPath(t *testing.T) {
	path, err := ResolveConfigPath("", filepath.Join("C:", "tools", "dvorakv.exe"))
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}
	want := filepath.Join("C:", "tools", "config.toml")
	if path != want {
		t.Errorf("ResolveConfigPath() = %q, want %q", path, want)
	}

	path, err = ResolveConfigPath("custom.toml", "")
	if err != nil {
		t.Fatalf("ResolveConfigPath() with explicit path error = %v", err)
	}
	if path != "custom.toml" {
		t.Errorf("ResolveConfigPath() = %q, want custom.toml", path)
	}
}
