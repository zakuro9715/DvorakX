package config

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

type dummyFormatter struct {
	err error
}

func (f dummyFormatter) FormatTo(writer io.Writer, errors []Error) error {
	var sb strings.Builder
	for _, err := range errors {
		if _, err := sb.WriteString(err.Message); err != nil {
			return err
		}
	}
	if f.err != nil {
		return f.err
	}
	_, err := io.WriteString(writer, sb.String())
	return err
}

func TestFormatTo(t *testing.T) {
	formatter := &dummyFormatter{}
	input := []Error{
		{Message: "first error"},
		{Message: "second error"},
	}
	var output bytes.Buffer
	if err := FormatTo(&output, formatter, input); err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}
	want := "first errorsecond error"
	if output.String() != want {
		t.Errorf("FormatTo() = %q, want %q", output.String(), want)
	}
}

func TestFormat(t *testing.T) {
	formatter := &dummyFormatter{}
	input := []Error{
		{Message: "error message"},
	}
	got, err := Format(formatter, input)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}
	want := "error message"
	if got != want {
		t.Errorf("Format() = %q, want %q", got, want)
	}
}

func TestFormatReturnsFormatterError(t *testing.T) {
	wantErr := errors.New("format failed")
	formatter := dummyFormatter{err: wantErr}
	_, err := Format(formatter, []Error{{Message: "message"}})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Format() error = %v, want %v", err, wantErr)
	}
}
