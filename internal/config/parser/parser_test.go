package parser

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestParseRemapsArrayMergesEntries(t *testing.T) {
	loaded, err := Parse([]byte(`
[global]
remaps = [
  { caps = "ctrl", a = "b" },
  { a = "c", b = "d" },
]
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assertDestination(t, loaded.Global, "a", "c")
	assertDestination(t, loaded.Global, "b", "d")
	assertDestination(t, loaded.Global, "caps", "ctrl")
}

func TestParseResolvesConfigurationStructure(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	if loaded.DefaultLayer != "base" || len(loaded.Layers) != 1 {
		t.Fatalf("unexpected layer selection: %#v", loaded)
	}
	assertDestination(t, loaded.Global, "caps", "ctrl")
	assertDestination(t, loaded.Layers["base"].Definition, "a", "a")
	assertDestination(t, loaded.Layers["base"].Definition, "b", "b")
	assertDestination(t, loaded.Layers["base"].Remaps, "x", "z")
}

func TestParseRemapsTable(t *testing.T) {
	loaded, err := Parse([]byte(`
[global.remaps]
caps = "ctrl"
a = "b"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assertDestination(t, loaded.Global, "a", "b")
	assertDestination(t, loaded.Global, "caps", "ctrl")
}

func TestParseRemapsTableSupportsNoopAction(t *testing.T) {
	loaded, err := Parse([]byte(`
[global.remaps]
x = "@disable"

[global.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := loaded.Global[key("x")]; got.Kind != config.NoopAction {
		t.Fatalf("binding = %#v, want noop", got)
	}
}

func TestParseKeepsBindingStagesSeparate(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	assertDestination(t, loaded.Layers["base"].Definition, "a", "b")
	assertDestination(t, loaded.Layers["base"].Remaps, "a", "c")
	assertDestination(t, loaded.Global, "a", "d")
}

func TestParseLayerRemapsUseLayerActions(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	if got := loaded.Layers["base"].Remaps[key("z")]; got.Kind != config.NoopAction {
		t.Fatalf("binding = %#v, want noop", got)
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
	loaded, err := Parse([]byte(`
[global]
remaps = [{ caps = "@disable" }]

[global.actions]
disable = { noop = [] }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := loaded.Global[key("caps")]; got.Kind != config.NoopAction {
		t.Fatalf("binding = %#v, want noop", got)
	}
}

func TestParseSendKeysActionReferencesSupportStringAndArray(t *testing.T) {
	loaded, err := Parse([]byte(`
[global]
remaps = [{ x = "@copy", y = "@paste" }]

[global.actions]
copy = { send-keys = ["ctrl", "c"] }
paste = { send-keys = "ctrl v" }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assertSequence(t, loaded.Global[key("x")], "ctrl", "c")
	assertSequence(t, loaded.Global[key("y")], "ctrl", "v")
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
	loaded, err := Parse([]byte(`
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
	if got := loaded.Layers["base"].Definition[key("a")]; got.Kind != config.NoopAction {
		t.Fatalf("binding = %#v, want noop", got)
	}
}

func TestParseLayerSendKeysActionReference(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	assertSequence(t, loaded.Layers["base"].Definition[key("a")], "ctrl", "c")
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
	issues, ok := validationIssues(err)
	if !ok || len(issues) != 1 {
		t.Fatalf("Parse() error = %v, want one structured error", err)
	}
	if got := issues[0]; got.Location().Text != "hotkey = 'abc'" || got.Location().Line != 2 || got.Location().Column != 1 {
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
	syntax, ok := errors.AsType[*SyntaxError](err)
	if !ok || errors.Unwrap(syntax) == nil {
		t.Fatalf("Parse() = %v, want syntax error with decoder cause", err)
	}
	if at := syntax.Location(); at.Line != 2 || at.Column < 1 || syntax.Error() == "" {
		t.Errorf("syntax error at %#v = %q, want line 2 and a message", at, syntax.Error())
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse([]byte("[global]\nremaps = [{ not-a-key = 'ctrl' }]\n"))
	if err == nil || !hasErrorMessage(err, `unsupported remap source key "not-a-key"`) {
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
	issues, ok := validationIssues(err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticReport", err)
	}
	if len(issues) != 3 {
		t.Fatalf("field error count = %d, want 3: %v", len(issues), issues)
	}
	for index, want := range []struct {
		column int
		text   string
	}{
		{column: 20, text: "invalid remap destination"},
		{column: 23, text: `unsupported remap source key "bad"`},
		{column: 44, text: `unsupported remap source key "nope"`},
	} {
		got := issues[index]
		if got.Location().Line != 2 || got.Location().Column != want.column {
			t.Errorf("diagnostic[%d] position = %d:%d, want 2:%d", index, got.Location().Line, got.Location().Column, want.column)
		}
		if !strings.Contains(got.Error(), want.text) {
			t.Errorf("diagnostic[%d] message = %q, want %q", index, got.Error(), want.text)
		}
	}
	if issues[0].Location().Column != 20 || issues[2].Location().Column != 44 {
		t.Errorf("errors = %#v, want all positions", issues)
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
	issues, ok := validationIssues(err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticReport", err)
	}
	if len(issues) != 4 {
		t.Fatalf("field error count = %d, want 4: %v", len(issues), issues)
	}
	for _, want := range []string{
		`unsupported source layer key "invalid"`,
		`unsupported layer key "also-invalid"`,
		`unsupported [other] field "extra"`,
		`[other].def is required`,
	} {
		found := false
		for _, diagnostic := range issues {
			if strings.Contains(diagnostic.Error(), want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("errors = %#v, want %q", issues, want)
		}
	}
}

func TestParseStaticDefaultLayer(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	definition := loaded.Layers["swap"].Definition
	assertDestination(t, definition, "a", "b")
	if len(definition) != 2 {
		t.Fatalf("definition = %#v, want two bindings", definition)
	}
}

func TestParseStaticDefaultLayerCanReuseSource(t *testing.T) {
	loaded, err := Parse([]byte(`
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
	assertDestination(t, loaded.Layers["qwerty"].Definition, "a", "a")
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
	issues, ok := validationIssues(err)
	if !ok {
		t.Fatalf("Parse() error type = %T, want *DiagnosticReport", err)
	}
	if len(issues) != 1 {
		t.Fatalf("len(errors) = %d, want 1: %#v", len(issues), issues)
	}
	got := issues[0]
	if got.Error() != `unsupported source layer key "invalid"` {
		t.Errorf("error message = %q, want source-layer error", got.Error())
	}
	if got.Location().Line != 6 || got.Location().Column != 8 {
		t.Errorf("error position = %d:%d, want 6:8", got.Location().Line, got.Location().Column)
	}
}

func TestParseBraceEscapedKeyDefinitions(t *testing.T) {
	loaded, err := Parse([]byte(`
[meta]
default = "&symbols"

[global]
remaps = [{ "{,}" = "{.}" }]

[src]
def = ["a", "b"]

[symbols]
def = "{,}|{'}"
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	assertDestination(t, loaded.Layers["symbols"].Definition, "a", ",")
	assertDestination(t, loaded.Layers["symbols"].Definition, "b", "'")
	assertDestination(t, loaded.Global, ",", ".")
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
			want: `unsupported layer key "invalid"`,
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
	issues, ok := validationIssues(err)
	if !ok {
		return false
	}
	for _, diagnostic := range issues {
		if strings.Contains(diagnostic.Error(), want) {
			return true
		}
	}
	return false
}

func key(name string) keys.Key {
	resolved, _ := keys.Lookup(name)
	return resolved
}

func assertDestination(t *testing.T, bindings config.Bindings, from, to string) {
	t.Helper()
	if got := bindings[key(from)]; got.Kind != config.RemapKey || got.Destination != key(to) {
		t.Errorf("binding for %q = %#v, want key %q", from, got, to)
	}
}

func assertSequence(t *testing.T, binding config.Binding, names ...string) {
	t.Helper()
	want := make([]keys.Key, len(names))
	for i, name := range names {
		want[i] = key(name)
	}
	if binding.Kind != config.SendKeysAction || !slices.Equal(binding.Keys, want) {
		t.Errorf("binding = %#v, want send-keys %v", binding, names)
	}
}

func TestParseValidatesNonDefaultDefinitions(t *testing.T) {
	_, err := Parse([]byte(`
[meta]
default = "&base"
[src]
def = ["a", "b"]
[base]
def = "#src"
[other]
def = ["invalid"]
`))
	for _, want := range []string{"different lengths", "unsupported layer key"} {
		if !hasErrorMessage(err, want) {
			t.Errorf("Parse() = %v, want %q", err, want)
		}
	}
}

func TestParseRejectsReservedDefaultAndDuplicateKeys(t *testing.T) {
	_, err := Parse([]byte(`
[meta]
default = "&global"
[src]
def = ["a", "A"]
[global.remaps]
a = "b"
A = "c"
`))
	for _, want := range []string{"undefined layer", "duplicate source layer key", "duplicate remap source key"} {
		if !hasErrorMessage(err, want) {
			t.Errorf("Parse() = %v, want %q", err, want)
		}
	}
}

func TestParseDiagnosticPathsDistinguishQuotedKeys(t *testing.T) {
	_, err := Parse([]byte(`
[global.actions]
"a.b" = { noop = [1] }
a.b = { noop = [] }
`))
	issues, ok := validationIssues(err)
	if !ok {
		t.Fatalf("Parse() = %v, want diagnostics", err)
	}
	for _, item := range issues {
		if strings.Contains(item.Error(), "does not accept arguments") {
			if item.Location().Line != 3 {
				t.Fatalf("diagnostic = %#v, want line 3", item)
			}
			return
		}
	}
	t.Fatal("missing diagnostic for quoted action name")
}

func TestParseRemapArrayTableOverwritesActions(t *testing.T) {
	loaded, err := Parse([]byte(`
[global.actions]
stop = { noop = [] }
[[global.remaps]]
a = "@stop"
[[global.remaps]]
A = "c"
`))
	if err != nil {
		t.Fatal(err)
	}
	assertDestination(t, loaded.Global, "a", "c")
}

func TestValidationErrorUnwrapReturnsIndependentCopy(t *testing.T) {
	_, err := Parse([]byte("[global]\nunknown = true\n"))
	validation, ok := errors.AsType[*ValidationError](err)
	if !ok {
		t.Fatalf("Parse() = %v, want validation error", err)
	}
	children := validation.Unwrap()
	children[0] = nil
	child, ok := errors.AsType[*FieldError](err)
	if !ok || child.Field() != "global.unknown" || child.Location().Text != "unknown = true" {
		t.Fatalf("field error = %#v, want global.unknown and its source line", child)
	}
}

func validationIssues(err error) ([]*FieldError, bool) {
	validation, ok := errors.AsType[*ValidationError](err)
	if !ok {
		return nil, false
	}
	issues := make([]*FieldError, 0)
	for _, child := range validation.Unwrap() {
		field, ok := errors.AsType[*FieldError](child)
		if !ok {
			return nil, false
		}
		issues = append(issues, field)
	}
	return issues, true
}
