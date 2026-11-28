package parser

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPrettyFormatterPrintsGroupedSourceContext(t *testing.T) {
	formatter := PrettyFormatter{}
	input := []Error{
		{Path: "config.toml", SourceLines: []string{"first", "bad = 1", "nearby", "farther", "bad = 2"}, Line: 2, Column: 1, Length: 3, Message: "first error"},
		{Path: "config.toml", SourceLines: []string{"first", "bad = 1", "nearby", "farther", "bad = 2"}, Line: 2, Column: 7, Length: 1, Message: "second error"},
		{Path: "config.toml", SourceLines: []string{"first", "bad = 1", "nearby", "farther", "bad = 2"}, Line: 5, Column: 1, Length: 3, Message: "third error"},
	}
	var output bytes.Buffer
	if err := formatter.FormatTo(&output, input); err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}
	for _, want := range []string{
		"[config.toml:2, 1]: first error",
		"[config.toml:2, 7]: second error",
		"   1 | first",
		"   2 | bad = 1",
		"   3 | nearby",
		"[config.toml:5, 1]: third error",
		"   4 | farther",
		"   5 | bad = 2",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("FormatTo() = %q, want %q", output.String(), want)
		}
	}
	if count := strings.Count(output.String(), "first error"); count != 1 {
		t.Errorf("FormatTo() contains first error %d times, want once: %q", count, output.String())
	}
}

func TestPrettyFormatterPrintsColors(t *testing.T) {
	formatter := PrettyFormatter{Color: true}
	var output bytes.Buffer
	if err := formatter.FormatTo(&output, []Error{{Line: 1, Column: 1, Length: 1, Message: "invalid", SourceLines: []string{"x"}}}); err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}
	if !strings.Contains(output.String(), "\x1b[") {
		t.Errorf("output = %q, want ANSI colors", output.String())
	}
}

func TestPrettyFormatterReturnsWriterError(t *testing.T) {
	formatter := PrettyFormatter{}
	wantErr := errors.New("write failed")
	err := formatter.FormatTo(errorWriter{err: wantErr}, []Error{{Message: "invalid"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("FormatTo() error = %v, want %v", err, wantErr)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
