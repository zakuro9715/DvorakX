package config

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	path, err := ResolveConfigPath("", filepath.Join("C:", "tools", "dvorakz.exe"))
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
