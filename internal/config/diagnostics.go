package config

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

// Diagnostics collects structured configuration errors.
type Diagnostics struct {
	Items []Error
}

// NewDiagnostics creates an empty diagnostics collection.
func NewDiagnostics() *Diagnostics {
	return &Diagnostics{}
}

// Add appends an error to the collection.
func (e *Diagnostics) Add(err Error) {
	e.Items = append(e.Items, err)
}

func (e *Diagnostics) Error() string {
	return fmt.Sprintf("there are %d configuration errors", len(e.Items))
}
