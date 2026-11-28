package parser

import (
	"fmt"
	"slices"
)

// Location identifies a span on one source line. Line and Column are one-based.
type Location struct {
	Path   string
	Line   int
	Column int
	Length int
	Text   string
}

// SyntaxError indicates that TOML decoding failed.
type SyntaxError struct {
	at      Location
	cause   error
	message string
}

// Error returns the reason TOML decoding failed.
func (e *SyntaxError) Error() string { return e.message }

// Location returns the source location of the decoding failure.
func (e *SyntaxError) Location() Location { return e.at }

// Unwrap returns the original decoder error.
func (e *SyntaxError) Unwrap() error { return e.cause }

// FieldError describes an invalid setting. Field is empty for document-wide failures.
type FieldError struct {
	at      Location
	field   string
	message string
}

// Error returns the reason validation failed.
func (e *FieldError) Error() string { return e.message }

// Field returns the path of the invalid setting.
func (e *FieldError) Field() string { return e.field }

// Location returns the source location of the invalid setting.
func (e *FieldError) Location() Location { return e.at }

// ValidationError groups invalid settings in source order.
type ValidationError struct {
	issues []error
}

// Error summarizes the validation failures.
func (e *ValidationError) Error() string {
	if len(e.issues) == 0 {
		return "invalid configuration"
	}
	return fmt.Sprintf("invalid configuration (%d issues): %s", len(e.issues), e.issues[0])
}

// Unwrap returns a copy of the list of field errors.
func (e *ValidationError) Unwrap() []error { return slices.Clone(e.issues) }
