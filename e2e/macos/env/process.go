//go:build darwin && cgo && e2e

package env

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// BuildRemapper builds the CLI used by the test.
func BuildRemapper(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "dvorakz")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", exe, "./cmd/dvorakz")
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	return exe
}

// StartRemapper starts the CLI and waits for its event tap to be installed.
func StartRemapper(t *testing.T, exe, configText string) {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(config, []byte(configText), 0o600))
	log, err := os.Create(filepath.Join(dir, "dvorakz.log"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	cmd := exec.Command(exe, "--config", config, "--debug")
	cmd.Stdout, cmd.Stderr = log, log
	require.NoError(t, cmd.Start())
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			_ = cmd.Process.Signal(syscall.SIGTERM)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				<-done
			}
		}
		data, err := os.ReadFile(log.Name())
		t.Logf("dvorakz exit: %v; log read: %v\n%s", waitErr, err, data)
	})

	deadline := time.Now().Add(10 * time.Second)
	ready := []byte("macOS keyboard event tap installed\n")
	for time.Now().Before(deadline) {
		select {
		case <-done:
			t.Fatalf("dvorakz exited before installing the macOS event tap: %v", waitErr)
		default:
		}
		data, err := os.ReadFile(log.Name())
		require.NoError(t, err)
		if bytes.Contains(data, ready) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the macOS remapper tap")
}
