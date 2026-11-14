package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
)

func TestRunPassesConfigurationAndDebugFlagToRunner(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[global]\nremaps = [{ a = \"b\" }]\n"), 0o600); err != nil {
		t.Fatalf("write configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := Run(ctx, []string{"dvorakx", "--config", configPath, "--debug"}, func(mappings *keyboard.Mappings, stop <-chan struct{}, debug bool) error {
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
