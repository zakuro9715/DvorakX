package configtests

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/cli"
	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

func TestConfigErrors(t *testing.T) {
	paths, err := filepath.Glob("*.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no TOML fixtures found")
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			input, want := readFixture(t, path)
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, path), []byte(input), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			err := cli.Run(context.Background(), []string{"dvorakz", "--config", path}, func(*keyboard.Mappings, <-chan struct{}, func() *keyboard.InputObserver, bool) error {
				t.Fatal("runner called for invalid configuration")
				return nil
			})
			if err == nil {
				t.Fatal("CLI accepted an invalid configuration")
			}
			if got := cli.FormatError(err, false); got != want {
				t.Errorf("unexpected error output\n--- want ---\n%s\n--- got ---\n%s\n--- end ---", want, got)
			}
		})
	}
}

func readFixture(t *testing.T, path string) (input, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	const marker = "# Output:"
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	divider := slices.Index(lines, marker)
	if divider < 0 {
		t.Fatalf("%s: missing %q marker", path, marker)
	}
	expected := lines[divider+1:]
	if slices.Contains(expected, marker) {
		t.Fatalf("%s: duplicate %q marker", path, marker)
	}
	for i, line := range expected {
		comment, ok := strings.CutPrefix(line, "# ")
		if !ok {
			t.Fatalf("%s:%d: expected output line must start with %q", path, divider+i+2, "# ")
		}
		expected[i] = comment
	}
	want = strings.Join(expected, "\n")
	if strings.TrimSpace(want) == "" {
		t.Fatalf("%s: expected output is empty", path)
	}
	if divider > 0 {
		input = strings.Join(lines[:divider], "\n") + "\n"
	}
	return input, want
}
