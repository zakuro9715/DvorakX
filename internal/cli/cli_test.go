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

func TestRunReturnsAllConfigurationDiagnostics(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nhotkeys = []\nremaps = [{ unknown = 'ctrl' }]\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	err := Run(context.Background(), []string{"dvorakz", "--config", configPath}, func(*keyboard.Mappings, <-chan struct{}, bool) error {
		t.Fatal("runner called for invalid configuration")
		return nil
	})
	if err == nil {
		t.Fatal("Run() error = nil, want configuration diagnostics")
	}
	diagnostics, ok := errors.AsType[*parser.DiagnosticsError](err)
	if !ok {
		t.Fatalf("Run() error type = %T, want *errors.DiagnosticsError", err)
	}
	for _, want := range []string{
		`unsupported [global] field "hotkeys"`,
		`unsupported remap source key "unknown"`,
	} {
		found := false
		for _, diagnostic := range diagnostics.Items {
			if strings.Contains(diagnostic.Message, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Run() error = %v, want %q", err, want)
		}
	}
}

func TestLoadConfigUsesSeparateDisplayPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nunknown = true\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	_, err := loadConfig(configPath, "config.toml")
	diagnostics, ok := errors.AsType[*parser.DiagnosticsError](err)
	if !ok || len(diagnostics.Items) != 1 {
		t.Fatalf("loadConfig() error = %v, want one diagnostic", err)
	}
	if got := diagnostics.Items[0].Path; got != "config.toml" {
		t.Errorf("diagnostic path = %q, want config.toml", got)
	}
}

func TestFormatConfigErrorPrefixesEveryDiagnostic(t *testing.T) {
	formatted := formatConfigError("reload", "config.toml", errors.New("line 2, column 3: first\nline 4, column 8: second"))
	want := "dvorakz: reload config \"config.toml\": line 2, column 3: first\n" +
		"dvorakz: reload config \"config.toml\": line 4, column 8: second"
	if formatted != want {
		t.Errorf("formatConfigError() = %q, want %q", formatted, want)
	}
	if strings.Count(formatted, "dvorakz: reload config") != 2 {
		t.Errorf("formatConfigError() = %q, want one prefix per diagnostic", formatted)
	}
}

func TestFormatConfigErrorRendersDiagnostics(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nunknown = true\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}
	_, err := parser.LoadFile(configPath)
	if err == nil {
		t.Fatal("Parse() error = nil, want diagnostic")
	}

	formatted := formatConfigError("reload", configPath, err)
	for _, want := range []string{
		fmt.Sprintf("dvorakz: reload config %q:", configPath),
		fmt.Sprintf("[%s:2, 1]: unsupported [global] field \"unknown\"", configPath),
		"   2 | unknown = true",
	} {
		if !strings.Contains(formatted, want) {
			t.Errorf("formatConfigError() = %q, want %q", formatted, want)
		}
	}
}
