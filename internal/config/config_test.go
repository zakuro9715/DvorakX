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

func TestParseRemapsTable(t *testing.T) {
	config, err := Parse([]byte(`
[global.remaps]
caps = "ctrl"
a = "b"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(config.Remap) != 2 {
		t.Fatalf("len(Remap) = %d, want 2", len(config.Remap))
	}
	if got := config.Remap[vkCapital]; got != vkControl {
		t.Errorf("caps remap = %#x, want %#x", got, vkControl)
	}
	if got := config.Remap[vkA]; got != vkA+1 {
		t.Errorf("a remap = %#x, want %#x", got, vkA+1)
	}
}

func TestParseRemapsTableSupportsNoopAction(t *testing.T) {
	config, err := Parse([]byte(`
[global.remaps]
x = "@disable"

[global.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !config.Noop[vkA+23] {
		t.Fatalf("Noop[x] = false, want true")
	}
}

func TestParseRejectsScalarRemaps(t *testing.T) {
	_, err := Parse([]byte(`
[global]
remaps = "invalid"
`))
	if err == nil || !hasErrorMessage(err, "[global].remaps must be a TOML table or table array") {
		t.Fatalf("Parse() error = %v, want invalid remaps type error", err)
	}
}

func TestParseGlobalNoopActionReference(t *testing.T) {
	config, err := Parse([]byte(`
[global]
remaps = [{ caps = "@disable" }]

[global.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !config.Noop[vkCapital] {
		t.Fatalf("Noop[caps] = false, want true")
	}
}

func TestParseLayerNoopActionReference(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&base"

[src]
def = ["a"]

[base]
def = ["@disable"]

[base.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if !config.Noop[vkA] {
		t.Fatalf("Noop[a] = false, want true")
	}
}

func TestParseRejectsUnknownAction(t *testing.T) {
	_, err := Parse([]byte(`
[global]
remaps = [{ caps = "@missing" }]
`))
	if err == nil || !hasErrorMessage(err, `undefined action "missing"`) {
		t.Fatalf("Parse() error = %v, want undefined action error", err)
	}
}

func TestParseFormatsDiagnosticWithSourceLine(t *testing.T) {
	_, err := Parse([]byte("[global]\nhotkey = 'abc'\n"))
	diagnostics, ok := err.(*Diagnostics)
	if !ok || len(diagnostics.Items) != 1 {
		t.Fatalf("Parse() error = %v, want one structured error", err)
	}
	if got := diagnostics.Items[0]; got.SourceLine != "hotkey = 'abc'" || got.Line != 2 || got.Column != 1 {
		t.Fatalf("error = %#v, want source and position", got)
	}
}

func TestParseCollectsUnsupportedGlobalFields(t *testing.T) {
	_, err := Parse([]byte("[global]\nhotkeys = []\naliases = []\n"))
	if err == nil {
		t.Fatal("Parse() error = nil, want unsupported field errors")
	}
	for _, want := range []string{
		`unsupported [global] field "hotkeys"`,
		`unsupported [global] field "aliases"`,
	} {
		if !hasErrorMessage(err, want) {
			t.Errorf("Parse() error = %v, want %q", err, want)
		}
	}
}

func TestParseReportsTomlSyntaxErrorPosition(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ caps = \"ctrl\" }\n"))
	diagnostics, ok := err.(*Diagnostics)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *Diagnostics", err)
	}
	if len(diagnostics.Items) != 1 {
		t.Fatalf("len(Diagnostics) = %d, want 1", len(diagnostics.Items))
	}
	diagnostic := diagnostics.Items[0]
	if diagnostic.Line != 2 || diagnostic.Column < 1 {
		t.Errorf("syntax diagnostic position = %d:%d, want line 2", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Message == "" {
		t.Errorf("syntax diagnostic message = %q, want a parser message", diagnostic.Message)
	}
	if diagnostic.Line != 2 || diagnostic.Column < 1 || diagnostic.Message == "" {
		t.Errorf("error = %#v, want line, column, and message", diagnostic)
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ capslock = 'ctrl' }]\n"))
	if err == nil || !hasErrorMessage(err, `unsupported remap source key "capslock"`) {
		t.Fatalf("Parse() error = %v, want unsupported key error", err)
	}
}

func TestParseRejectsInvalidRemapValue(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ caps = 1 }]\n"))
	if err == nil || !hasErrorMessage(err, "invalid remap destination") {
		t.Fatalf("Parse() error = %v, want remap destination error", err)
	}
}

func TestParseCollectsAllRemapErrorsWithPositions(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ caps = 1, bad = 'invalid' }, { nope = 'ctrl' }]\n"))
	diagnostics, ok := err.(*Diagnostics)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *Diagnostics", err)
	}
	if len(diagnostics.Items) != 3 {
		t.Fatalf("len(Diagnostics) = %d, want 3: %v", len(diagnostics.Items), diagnostics)
	}
	for index, want := range []struct {
		column int
		text   string
	}{
		{column: 13, text: "invalid remap destination"},
		{column: 23, text: `unsupported remap source key "bad"`},
		{column: 44, text: `unsupported remap source key "nope"`},
	} {
		got := diagnostics.Items[index]
		if got.Line != 2 || got.Column != want.column {
			t.Errorf("diagnostic[%d] position = %d:%d, want 2:%d", index, got.Line, got.Column, want.column)
		}
		if !strings.Contains(got.Message, want.text) {
			t.Errorf("diagnostic[%d] message = %q, want %q", index, got.Message, want.text)
		}
	}
	if diagnostics.Items[0].Column != 13 || diagnostics.Items[2].Column != 44 {
		t.Errorf("errors = %#v, want all positions", diagnostics.Items)
	}
}

func TestParseCollectsIndependentLayerErrors(t *testing.T) {
	_, err := Parse([]byte(`
[meta]
default = "&default"

[src]
def = ["invalid"]

[default]
def = ["also-invalid"]

[other]
extra = true
`))
	diagnostics, ok := err.(*Diagnostics)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *Diagnostics", err)
	}
	if len(diagnostics.Items) != 3 {
		t.Fatalf("len(Diagnostics) = %d, want 3: %v", len(diagnostics.Items), diagnostics)
	}
	for _, want := range []string{
		`unsupported source layer key "invalid"`,
		`unsupported [other] field "extra"`,
		`[other].def is required`,
	} {
		found := false
		for _, diagnostic := range diagnostics.Items {
			if strings.Contains(diagnostic.Message, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("errors = %#v, want %q", diagnostics.Items, want)
		}
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

func TestParseStaticDefaultLayerReportsInvalidSourceOnce(t *testing.T) {
	_, err := Parse([]byte(`
[meta]
default = "&qwerty"

[src]
def = ["invalid"]

[qwerty]
def = "#src"
`))
	diagnostics, ok := err.(*Diagnostics)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *errors.Diagnostics", err)
	}
	if len(diagnostics.Items) != 1 {
		t.Fatalf("len(errors) = %d, want 1: %#v", len(diagnostics.Items), diagnostics.Items)
	}
	got := diagnostics.Items[0]
	if got.Message != `unsupported source layer key "invalid" at position 0` {
		t.Errorf("error message = %q, want source-layer error", got.Message)
	}
	if got.Line != 6 || got.Column != 8 {
		t.Errorf("error position = %d:%d, want 6:8", got.Line, got.Column)
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
			if err == nil || !hasErrorMessage(err, test.want) {
				t.Fatalf("Parse() error = %v, want %q", err, test.want)
			}
		})
	}
}

func hasErrorMessage(err error, want string) bool {
	diagnostics, ok := err.(*Diagnostics)
	if !ok {
		return false
	}
	for _, diagnostic := range diagnostics.Items {
		if strings.Contains(diagnostic.Message, want) {
			return true
		}
	}
	return false
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
