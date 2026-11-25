package config

import (
	"errors"
	"path/filepath"
	"reflect"
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
	if len(config.Global.Remaps) != 2 || config.Global.Remaps[1]["a"] != "c" {
		t.Errorf("Global.Remaps = %#v, want ordered raw remap tables", config.Global.Remaps)
	}
}

func TestParsePreservesConfigurationStructure(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&base"

[global]
remaps = [{ caps = "ctrl" }]

[src]
def = ["a", "b"]

[base]
def = "#src"

[base.remaps]
x = "z"

[base.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if config.Meta.Default != "&base" {
		t.Errorf("Meta.Default = %q, want &base", config.Meta.Default)
	}
	if len(config.Global.Remaps) != 1 || config.Global.Remaps[0]["caps"] != "ctrl" {
		t.Errorf("Global.Remaps = %#v, want the global remap table", config.Global.Remaps)
	}
	if got := config.Source.Definition.Keys; len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("Source.Definition.Keys = %#v, want [a b]", got)
	}
	base, ok := config.Layers["base"]
	if !ok {
		t.Fatal("Layers[base] is missing")
	}
	if base.Definition.Text != "#src" {
		t.Errorf("base definition = %q, want #src", base.Definition.Text)
	}
	if base.Remaps[0]["x"] != "z" {
		t.Errorf("base remaps = %#v, want x -> z", base.Remaps)
	}
	if base.Actions["disable"].Kind != NoopAction {
		t.Errorf("base actions = %#v, want disable=noop", base.Actions)
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
	if len(config.Global.Remaps) != 1 || config.Global.Remaps[0]["a"] != "b" {
		t.Errorf("Global.Remaps = %#v, want raw remap table", config.Global.Remaps)
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
	if got := config.Global.Remaps[0]["x"]; got != "@disable" {
		t.Fatalf("global remap x = %q, want @disable", got)
	}
}

func TestParseLayerRemapsApplyAfterDefinitionAndBeforeGlobal(t *testing.T) {
	config, err := Parse([]byte(`
[global.remaps]
a = "d"

[meta]
default = "&base"

[src]
def = ["a", "b", "c"]

[base]
def = ["b", "c", "a"]
remaps = [
  { a = "c" },
	{ x = "z" },
  { z = "x" },
]
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(config.Layers["base"].Remaps) != 3 {
		t.Fatalf("layer remaps = %#v, want three ordered definitions", config.Layers["base"].Remaps)
	}
}

func TestParseLayerRemapsUseLayerActions(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&base"

[src]
def = ["a"]

[base]
def = ["a"]

[base.remaps]
z = "@disable"

[base.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Layers["base"].Remaps[0]["z"]; got != "@disable" {
		t.Fatalf("layer remap z = %q, want @disable", got)
	}
}

func TestParseNonDefaultLayerRemapsAreValidated(t *testing.T) {
	_, err := Parse([]byte(`
[meta]
default = "&base"

[src]
def = ["a"]

[base]
def = ["a"]

[other]
def = ["a"]

[other.remaps]
invalid = "ctrl"
`))
	if err == nil || !hasErrorMessage(err, `unsupported remap source key "invalid"`) {
		t.Fatalf("Parse() error = %v, want unsupported layer remap key error", err)
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
	if got := config.Global.Remaps[0]["caps"]; got != "@disable" {
		t.Fatalf("global remap caps = %q, want @disable", got)
	}
}

func TestParseSendKeysActionReferencesSupportStringAndArray(t *testing.T) {
	config, err := Parse([]byte(`
[global]
remaps = [{ x = "@copy", y = "@paste" }]

[global.actions]
copy = { send-keys = ["ctrl", "c"] }
paste = { send-keys = "ctrl v" }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Global.Remaps[0]["x"]; got != "@copy" {
		t.Errorf("global remap x = %q, want @copy", got)
	}
	if got := config.Global.Actions["paste"].Kind; got != SendKeysAction {
		t.Errorf("paste action kind = %v, want send-keys", got)
	}
}

func TestParseRejectsInvalidSendKeysAction(t *testing.T) {
	tests := []struct {
		name string
		data string
		want string
	}{
		{
			name: "invalid argument type",
			data: "[global.actions]\ncopy = { send-keys = 1 }\n[global.remaps]\nx = '@copy'\n",
			want: "send-keys must be a string or string array",
		},
		{
			name: "empty argument",
			data: "[global.actions]\ncopy = { send-keys = '' }\n[global.remaps]\nx = '@copy'\n",
			want: "send-keys must contain at least one key",
		},
		{
			name: "unknown key",
			data: "[global.actions]\ncopy = { send-keys = ['unknown'] }\n[global.remaps]\nx = '@copy'\n",
			want: `unsupported send-keys key "unknown"`,
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
	if got := config.Layers["base"].Definition.Keys[0]; got != "@disable" {
		t.Fatalf("layer definition = %q, want @disable", got)
	}
}

func TestParseLayerSendKeysActionReference(t *testing.T) {
	config, err := Parse([]byte(`
[meta]
default = "&base"

[src]
def = ["a"]

[base]
def = ["@copy"]

[base.actions]
copy = { send-keys = "ctrl c" }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := config.Layers["base"].Definition.Keys[0]; got != "@copy" {
		t.Errorf("layer definition = %q, want @copy", got)
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticsError", err)
	}
	if len(diagnostics.Items) != 1 {
		t.Fatalf("len(DiagnosticsError) = %d, want 1", len(diagnostics.Items))
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticsError", err)
	}
	if len(diagnostics.Items) != 3 {
		t.Fatalf("len(DiagnosticsError) = %d, want 3: %v", len(diagnostics.Items), diagnostics)
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticsError", err)
	}
	if len(diagnostics.Items) != 3 {
		t.Fatalf("len(DiagnosticsError) = %d, want 3: %v", len(diagnostics.Items), diagnostics)
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
	if got := config.Layers["swap"].Definition.Tokens; !reflect.DeepEqual(got, []string{"b", "_", "a"}) {
		t.Errorf("swap definition tokens = %#v, want [b _ a]", got)
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
	if got := config.Layers["qwerty"].Definition.Tokens; !reflect.DeepEqual(got, []string{"#src"}) {
		t.Errorf("qwerty definition tokens = %#v, want [#src]", got)
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *errors.DiagnosticsError", err)
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
	if got := config.Layers["symbols"].Definition.Tokens; !reflect.DeepEqual(got, []string{",", "|", "'"}) {
		t.Errorf("symbols definition tokens = %#v, want escaped key tokens", got)
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
	diagnostics, ok := errors.AsType[*DiagnosticsError](err)
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
