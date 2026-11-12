package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRemapsArrayMergesEntries(t *testing.T) {
	config, err := Parse([]byte(`
[global]
remaps = [
  { caps = "ctrl", a = "b" },
  { a = "c", b = "d" },
]
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(config.Remap) != 3 {
		t.Fatalf("len(Remap) = %d, want 3", len(config.Remap))
	}
	if got := config.Remap[vkCapital]; got != vkControl {
		t.Errorf("caps remap = %#x, want %#x", got, vkControl)
	}
	if got := config.Remap[vkA]; got != vkA+2 {
		t.Errorf("a remap = %#x, want %#x", got, vkA+2)
	}
	if got := config.Remap[vkA+1]; got != vkA+3 {
		t.Errorf("b remap = %#x, want %#x", got, vkA+3)
	}
}

func TestParseRejectsUnsupportedConfiguration(t *testing.T) {
	_, err := Parse([]byte("[global]\norder = ['qwerty']\n"))
	if err == nil || !strings.Contains(err.Error(), `unsupported [global] field "order"`) {
		t.Fatalf("Parse() error = %v, want unsupported field error", err)
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ capslock = 'ctrl' }]\n"))
	if err == nil || !strings.Contains(err.Error(), `unsupported remap source key "capslock"`) {
		t.Fatalf("Parse() error = %v, want unsupported key error", err)
	}
}

func TestParseRejectsInvalidRemapValue(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ caps = 1 }]\n"))
	if err == nil || !strings.Contains(err.Error(), "invalid remap destination") {
		t.Fatalf("Parse() error = %v, want remap destination error", err)
	}
}

func TestParseStaticDefaultLayer(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&swap"

[global]
remaps = [{ a = "c" }]

[src]
def = ["a", "b", "c"]

[swap]
def = """
# The separators below are all accepted.
b | _ , a
"""
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Remap[vkA]; got != vkA+2 {
		t.Errorf("a remap = %#x, want %#x", got, vkA+2)
	}
	if _, ok := config.Remap[vkA+1]; ok {
		t.Error("b remap exists, want placeholder to leave it unmapped")
	}
	if got := config.Remap[vkA+2]; got != vkA {
		t.Errorf("c remap = %#x, want %#x", got, vkA)
	}
}

func TestParseStaticDefaultLayerCanReuseSource(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&qwerty"

[src]
def = ["a", "b"]

[qwerty]
def = "#src"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Remap[vkA]; got != vkA {
		t.Errorf("a remap = %#x, want %#x", got, vkA)
	}
	if got := config.Remap[vkA+1]; got != vkA+1 {
		t.Errorf("b remap = %#x, want %#x", got, vkA+1)
	}
}

func TestParseBraceEscapedKeyDefinitions(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&symbols"

[global]
remaps = [{ "{,}" = "{|}" }]

[src]
def = ["a", "b", "c"]

[symbols]
def = "{,}|{|}|{'}"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Remap[vkA]; got != vkOemComma {
		t.Errorf("a remap = %#x, want %#x", got, vkOemComma)
	}
	if got := config.Remap[vkA+1]; got != vkOem5 {
		t.Errorf("b remap = %#x, want %#x", got, vkOem5)
	}
	if got := config.Remap[vkA+2]; got != vkOem7 {
		t.Errorf("c remap = %#x, want %#x", got, vkOem7)
	}
	if got := config.Remap[vkOemComma]; got != vkOem5 {
		t.Errorf("comma remap = %#x, want %#x", got, vkOem5)
	}
}

func TestParseRejectsMalformedBraceEscapes(t *testing.T) {
	for _, definition := range []string{"{,", "}", "{}", "{a{}}"} {
		t.Run(definition, func(t *testing.T) {
			_, err := Parse([]byte("[meta]\ndefault = '&layer'\n[src]\ndef = ['a']\n[layer]\ndef = '" + definition + "'\n"))
			if err == nil {
				t.Fatalf("Parse() error = %v, want brace error", err)
			}
		})
	}
}

func TestParseRejectsInvalidStaticLayer(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "missing default layer",
			data: "[src]\ndef = ['a']\n",
			want: "requires [meta]",
		},
		{
			name: "legacy layer reference prefix",
			data: "[meta]\ndefault = '@layer'\n[src]\ndef = ['a']\n[layer]\ndef = ['a']\n",
			want: "reference a layer with &",
		},
		{
			name: "undefined default layer",
			data: "[meta]\ndefault = '&missing'\n[src]\ndef = ['a']\n",
			want: `undefined layer "missing"`,
		},
		{
			name: "mismatched definitions",
			data: "[meta]\ndefault = '&layer'\n[src]\ndef = ['a']\n[layer]\ndef = ['a', 'b']\n",
			want: "different lengths",
		},
		{
			name: "invalid destination key",
			data: "[meta]\ndefault = '&layer'\n[src]\ndef = ['a']\n[layer]\ndef = ['invalid']\n",
			want: `unsupported default layer key "invalid"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.data))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Parse() error = %v, want %q", err, test.want)
			}
		})
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
