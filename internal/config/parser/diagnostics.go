package parser

import "fmt"

// Error describes a single configuration validation failure.
type Error struct {
	Line        int
	Column      int
	Length      int
	SourceLine  string
	SourceLines []string
	Path        string
	Message     string
}

// DiagnosticsError collects structured configuration errors.
type DiagnosticsError struct {
	Items []Error
}

// NewDiagnostics creates an empty diagnostics collection.
func NewDiagnostics() *DiagnosticsError {
	return &DiagnosticsError{}
}

// Add appends an error to the collection.
func (e *DiagnosticsError) Add(err Error) {
	e.Items = append(e.Items, err)
}

func (e *DiagnosticsError) Error() string {
	return fmt.Sprintf("there are %d configuration errors", len(e.Items))
}
