package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	for _, mode := range []string{"set", "count", "atomic"} {
		t.Run(mode, func(t *testing.T) {
			const blocks = `example/module2/z.go:3.2,3.4 1 0
example/module/a.go:2.1,4.1 2 1
example/module/a.go:2.1,4.1 2 1
example/module/a.go:3.2,3.5 1 1
example/module/a.go:5.1,5.2 0 1
`
			count := 2
			if mode == "set" {
				count = 1
			}
			want := fmt.Sprintf("SF:a.go\nDA:2,%d\nDA:3,%d\nLF:2\nLH:2\nend_of_record\n"+
				"SF:example/module2/z.go\nDA:3,0\nLF:1\nLH:0\nend_of_record\n", count, count)
			var got bytes.Buffer
			if err := convert(strings.NewReader("mode: "+mode+"\n"+blocks), &got, "example/module", ""); err != nil {
				t.Fatal(err)
			}
			if got.String() != want {
				t.Fatalf("got:\n%s\nwant:\n%s", got.String(), want)
			}
		})
	}
}

func TestInvalidProfile(t *testing.T) {
	for _, profile := range []string{"", "mode: invalid", "mode: set\ninvalid", "mode: set\na.go:3.2,2.1 1 0"} {
		var got bytes.Buffer
		if err := convert(strings.NewReader(profile), &got, "", ""); err == nil || got.Len() != 0 {
			t.Errorf("convert(%q) produced %q and error %v", profile, got.String(), err)
		}
	}
}

func TestRunFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage profile.out")
	if err := os.WriteFile(path, []byte("mode: atomic\r\npkg/a.go:7.2,7.9 1 4\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(t.TempDir(), "report.info")
	if err := run([]string{path, output}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	const want = "SF:pkg/a.go\nDA:7,4\nLF:1\nLH:1\nend_of_record\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRunArguments(t *testing.T) {
	for _, args := range [][]string{nil, {"input"}, {"input", "output", "extra"}} {
		if err := run(args); err == nil || !strings.Contains(err.Error(), "gocov2lcov in_gocov out_lcov") {
			t.Errorf("run(%q) returned %v, want usage error", args, err)
		}
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestConvertWriteFailure(t *testing.T) {
	input := strings.NewReader("mode: set\na.go:1.2,1.4 1 1\n")
	if err := convert(input, brokenWriter{}, "", ""); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("got error %v, want %v", err, io.ErrClosedPipe)
	}
}

func TestRunResolvesModulePaths(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.org/project\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "sub")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(subdir)
	input := filepath.Join(root, "coverage.out")
	if err := os.WriteFile(input, []byte("mode: set\nexample.org/project/sub/main.go:1.2,1.4 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(root, "report.info")
	if err := run([]string{input, output}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	absolute, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	want := "SF:" + filepath.ToSlash(filepath.Join(absolute, "main.go")) + "\nDA:1,1\nLF:1\nLH:1\nend_of_record\n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
