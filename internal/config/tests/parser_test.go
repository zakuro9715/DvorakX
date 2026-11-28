package configtests

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/config/parser"
)

func TestParseCRLFDefinition(t *testing.T) {
	input := "[meta]\r\ndefault = '&base'\r\n[src]\r\ndef = 'a b'\r\n[base]\r\ndef = \"\"\"\r\n a\r\n invalid-key\r\n\"\"\"\r\n"
	_, err := parser.Parse([]byte(input))
	field, ok := errors.AsType[*parser.FieldError](err)
	if !ok {
		t.Fatalf("Parse() = %v, want field error", err)
	}
	want := parser.Location{Line: 8, Column: 2, Length: 11, Text: " invalid-key"}
	if got := field.Location(); got != want {
		t.Errorf("Location() = %#v, want %#v", got, want)
	}
}

func TestLoadFileMissingConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	if _, err := parser.LoadFile(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadFile() = %v, want missing file error", err)
	}
}

func TestValidationErrorSummary(t *testing.T) {
	_, err := parser.Parse([]byte("[global]\nhotkeys = []\nremaps = [{ unknown = 'ctrl' }]\n"))
	validation, ok := errors.AsType[*parser.ValidationError](err)
	if !ok || len(validation.Unwrap()) != 2 {
		t.Fatalf("Parse() = %v, want two validation errors", err)
	}
	want := `invalid configuration (2 issues): unsupported [global] field "hotkeys"`
	if got := validation.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
