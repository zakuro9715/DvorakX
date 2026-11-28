package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zakuro9715/DvorakZ/internal/config/parser"
	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

func TestRunPassesConfigurationAndDebugFlagToRunner(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nremaps = [{ a = \"b\" }]\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := Run(ctx, []string{"dvorakz", "--config", configPath, "--debug"}, func(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error {
		if !debug {
			t.Error("debug = false, want true")
		}
		if got := len(mappings.Load()); got != 1 {
			t.Errorf("len(mappings.Load()) = %d, want 1", got)
		}
		cancel()
		select {
		case <-stop:
		case <-time.After(time.Second):
			t.Fatal("stop channel was not closed after context cancellation")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestLoadConfigUsesSeparateDisplayPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nunknown = true\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	_, err := loadConfig(configPath, "config.toml")
	validation, ok := errors.AsType[*parser.ValidationError](err)
	if !ok || len(validation.Unwrap()) != 1 {
		t.Fatalf("loadConfig() error = %v, want one diagnostic", err)
	}
	field, ok := errors.AsType[*parser.FieldError](err)
	if !ok || field.Location().Path != configPath {
		t.Fatalf("field error = %#v, want actual path %q", field, configPath)
	}
	want := `[config.toml:2, 1]: unsupported [global] field "unknown"
   2 | unknown = true
     | ^^^^^^^`
	assertErrorOutput(t, FormatError(err, false), want)
}

func TestFormatReloadErrorPrefixesEachLine(t *testing.T) {
	err := errors.New(`read failed
file unavailable`)
	want := `dvorakz: reload config "config.toml": read failed
dvorakz: reload config "config.toml": file unavailable`
	assertErrorOutput(t, formatReloadError("config.toml", err), want)
}

func TestFormatReloadErrorRendersValidationError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nunknown = true\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}
	_, err := parser.LoadFile(configPath)
	if err == nil {
		t.Fatal("Parse() error = nil, want diagnostic")
	}

	want := fmt.Sprintf(`dvorakz: reload config %q:
[%s:2, 1]: unsupported [global] field "unknown"
   2 | unknown = true
     | ^^^^^^^`, configPath, configPath)
	assertErrorOutput(t, formatReloadError(configPath, err), want)
}

func TestReloadConfigKeepsActiveMappingWhenInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	mappings := keyboard.NewMappings(nil)
	input, _ := keys.Lookup("a")
	write := func(destination string) {
		t.Helper()
		data := fmt.Sprintf("[global.remaps]\na = %q\n", destination)
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Load a valid mapping, then verify a failed reload keeps it active.
	write("b")
	if err := reloadConfig(path, "", mappings); err != nil {
		t.Fatal(err)
	}
	write("invalid-key")
	if err := reloadConfig(path, "", mappings); err == nil {
		t.Fatal("reload succeeded with an invalid key")
	}
	want, _ := keys.Lookup("b")
	if got, ok := mappings.Resolve(input); !ok || got.Destination != want {
		t.Fatalf("mapping after failed reload = %#v, want b", got)
	}
}

func TestFormatErrorPreservesReadErrorAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	_, err := loadConfig(path, "config.toml")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("loadConfig() = %v, want missing file error", err)
	}
	if got := FormatError(err, false); !strings.Contains(got, path) {
		t.Errorf("FormatError() = %q, want actual path %q", got, path)
	}
}

func TestFormatReloadErrorRendersSyntaxLocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("[global]\nremaps = ]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadConfig(path, "config.toml")
	syntax, ok := errors.AsType[*parser.SyntaxError](err)
	if !ok || syntax.Location().Path != path {
		t.Fatalf("loadConfig() = %v, want syntax error with actual path", err)
	}
	want := fmt.Sprintf(`dvorakz: reload config %q:
[config.toml:2, 10]: unexpected character U+005D ']' at start of value
   2 | remaps = ]
     |          ^`, path)
	rendered := formatReloadError(path, err, true)
	const red, reset = "\x1b[31m", "\x1b[0m"
	if strings.Count(rendered, red) != 2 || strings.Count(rendered, reset) != 2 {
		t.Errorf("expected ANSI color around heading and caret, got %q", rendered)
	}
	assertErrorOutput(t, strings.NewReplacer(red, "", reset, "").Replace(rendered), want)
}

func assertErrorOutput(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("unexpected error output\n--- want ---\n%s\n--- got ---\n%s\n--- end ---", want, got)
	}
}
