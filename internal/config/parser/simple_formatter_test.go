package parser

import (
	"bytes"
	"errors"
	"testing"
)

func TestSimpleFormatterFormatsErrorsInInputOrder(t *testing.T) {
	var output bytes.Buffer
	errs := []Error{
		{Path: "config.toml", Line: 1, Column: 1, Message: "first"},
		{Path: "config.toml", Line: 1, Column: 2, Message: "second"},
	}
	if err := (SimpleFormatter{}).FormatTo(&output, errs); err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}
	want := "config.toml:1:1: first\nconfig.toml:1:2: second"
	if output.String() != want {
		t.Errorf("FormatTo() = %q, want %q", output.String(), want)
	}
}

func TestSimpleFormatterLeavesEmptyPathEmpty(t *testing.T) {
	var output bytes.Buffer
	if err := (SimpleFormatter{}).FormatTo(&output, []Error{{Line: 2, Column: 3, Message: "message"}}); err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}
	if got, want := output.String(), ":2:3: message"; got != want {
		t.Errorf("FormatTo() = %q, want %q", got, want)
	}
}

func TestSimpleFormatterReturnsWriterError(t *testing.T) {
	wantErr := errors.New("write failed")
	err := (SimpleFormatter{}).FormatTo(errorWriter{err: wantErr}, []Error{{Message: "message"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("FormatTo() error = %v, want %v", err, wantErr)
	}
}
