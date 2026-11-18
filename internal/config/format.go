package config

import (
	"bytes"
	"io"
)

// Formatter writes diagnostics in a user-facing format.
type Formatter interface {
	FormatTo(io.Writer, []Error) error
}

// FormatTo writes errors with formatter.
func FormatTo(writer io.Writer, formatter Formatter, errors []Error) error {
	return formatter.FormatTo(writer, errors)
}

// Format returns the formatted errors as a string.
func Format(formatter Formatter, errors []Error) (string, error) {
	var buf bytes.Buffer
	err := formatter.FormatTo(&buf, errors)
	return buf.String(), err
}
