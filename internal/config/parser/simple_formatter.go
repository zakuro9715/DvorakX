package parser

import (
	"fmt"
	"io"
	"strings"
)

// SimpleFormatter renders one compact line for each error.
type SimpleFormatter struct{}

// FormatTo writes errors as path:line:column: message.
func (SimpleFormatter) FormatTo(writer io.Writer, errors []Error) error {
	if len(errors) == 0 {
		return nil
	}

	var output strings.Builder
	for index, err := range errors {
		if index > 0 {
			output.WriteByte('\n')
		}
		fmt.Fprintf(&output, "%s:%d:%d: %s", err.Path, err.Line, err.Column, err.Message)
	}
	_, writeErr := io.WriteString(writer, output.String())
	return writeErr
}
