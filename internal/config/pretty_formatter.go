package config

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

const (
	colorRed    = "\x1b[31m"
	colorYellow = "\x1b[33m"
	colorCyan   = "\x1b[36m"
)

// PrettyFormatter renders errors with source context and optional ANSI colors.
type PrettyFormatter struct {
	Color bool
}

// FormatTo formats errors to writer.
func (p PrettyFormatter) FormatTo(writer io.Writer, errors []Error) error {
	if len(errors) == 0 {
		return nil
	}

	entries := append([]Error(nil), errors...)
	sort.SliceStable(entries, func(left, right int) bool {
		if entries[left].Line != entries[right].Line {
			return entries[left].Line < entries[right].Line
		}
		return entries[left].Column < entries[right].Column
	})

	var output strings.Builder
	for groupIndex, group := range groupErrors(entries) {
		if groupIndex > 0 {
			output.WriteString("\n")
		}
		path := group[0].Path
		for _, err := range group {
			fmt.Fprintf(&output, "%s: %s\n",
				style(p.Color, colorYellow, fmt.Sprintf("[%s:%d, %d]", path, err.Line, err.Column)),
				style(p.Color, colorRed, err.Message))
		}

		startLine, endLine := sourceRange(group, len(group[0].SourceLines))
		for line := startLine; line <= endLine; line++ {
			source := sourceContextLine(group[0].SourceLines, line, group)
			fmt.Fprintf(&output, "%s\n",
				style(p.Color, colorCyan, fmt.Sprintf("%4d | %s", line, source)))
			for _, err := range group {
				if err.Line != line {
					continue
				}
				marker := strings.Repeat(" ", nonNegative(err.Column-1)) +
					strings.Repeat("^", atLeastOne(err.Length))
				fmt.Fprintf(&output, "     | %s\n",
					style(p.Color, colorRed, marker))
			}
		}
	}

	_, err := io.WriteString(writer, strings.TrimSuffix(output.String(), "\n"))
	return err
}

func groupErrors(errors []Error) [][]Error {
	var groups [][]Error
	for _, err := range errors {
		if len(groups) == 0 ||
			err.Path != groups[len(groups)-1][0].Path ||
			err.Line > groups[len(groups)-1][len(groups[len(groups)-1])-1].Line+1 {
			groups = append(groups, nil)
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], err)
	}
	return groups
}

func sourceRange(errors []Error, sourceLineCount int) (int, int) {
	start := atLeastOne(errors[0].Line - 1)
	end := errors[len(errors)-1].Line + 1
	if sourceLineCount > 0 {
		end = atMost(end, sourceLineCount)
	}
	return start, end
}

func sourceContextLine(sourceLines []string, line int, errors []Error) string {
	if line >= 1 && line <= len(sourceLines) {
		return sourceLines[line-1]
	}
	for _, err := range errors {
		if err.Line == line {
			return err.SourceLine
		}
	}
	return ""
}

func style(enabled bool, code, text string) string {
	if !enabled {
		return text
	}
	return code + text + "\x1b[0m"
}

func nonNegative(value int) int {
	return atLeast(value, 0)
}

func atLeastOne(value int) int {
	return atLeast(value, 1)
}

func atLeast(value, minimum int) int {
	if value < minimum {
		return minimum
	}
	return value
}

func atMost(value, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}
