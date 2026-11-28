package cli

import (
	"path/filepath"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	path, err := resolveConfigPath("", filepath.Join("C:", "tools", "dvorakz.exe"))
	if err != nil {
		t.Fatalf("resolveConfigPath() error = %v", err)
	}
	want := filepath.Join("C:", "tools", "config.toml")
	if path != want {
		t.Errorf("resolveConfigPath() = %q, want %q", path, want)
	}

	path, err = resolveConfigPath("custom.toml", "")
	if err != nil {
		t.Fatalf("resolveConfigPath() with explicit path error = %v", err)
	}
	if path != "custom.toml" {
		t.Errorf("resolveConfigPath() = %q, want custom.toml", path)
	}
}
