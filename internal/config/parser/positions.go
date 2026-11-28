package parser

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/pelletier/go-toml/v2/unstable"
)

type sourcePosition struct {
	line    int
	column  int
	length  int
	value   *sourcePosition
	content *sourcePosition
	brace   *sourcePosition
}

type sourcePositions map[string]sourcePosition

func collectSourcePositions(data []byte) (sourcePositions, error) {
	positions := make(sourcePositions)
	parser := &unstable.Parser{}
	parser.Reset(data)
	tablePath := ""
	arrayCounts := make(map[string]int)
	for parser.NextExpression() {
		expression := parser.Expression()
		switch expression.Kind {
		case unstable.Table, unstable.ArrayTable:
			tablePath = ""
			key := expression.Key()
			for key.Next() {
				node := key.Node()
				tablePath = joinPath(tablePath, string(node.Data))
				position := nodePosition(parser, node)
				if _, exists := positions[tablePath]; !exists {
					positions[tablePath] = position
				}
				if key.IsLast() && expression.Kind == unstable.ArrayTable {
					index := arrayCounts[tablePath]
					arrayCounts[tablePath]++
					tablePath = indexPath(tablePath, index)
					positions[tablePath] = position
				} else if count := arrayCounts[tablePath]; count > 0 {
					tablePath = indexPath(tablePath, count-1)
				}
			}
		case unstable.KeyValue:
			recordNodePositions(parser, expression, tablePath, positions)
		case unstable.Invalid, unstable.Comment, unstable.Key, unstable.Array, unstable.InlineTable,
			unstable.String, unstable.Bool, unstable.Float, unstable.Integer,
			unstable.LocalDate, unstable.LocalTime, unstable.LocalDateTime, unstable.DateTime:
		}
	}
	if err := parser.Error(); err != nil {
		return nil, err
	}
	return positions, nil
}

func recordNodePositions(parser *unstable.Parser, expression *unstable.Node, tablePath string, positions sourcePositions) {
	key := expression.Key()
	if !key.Next() {
		return
	}
	recordKeyValuePositions(parser, expression, tablePath, key, positions)
}

func recordKeyValuePositions(parser *unstable.Parser, expression *unstable.Node, parent string, key unstable.Iterator, positions sourcePositions) int {
	path := appendPath(parent, keyPath(expression))
	position := nodePosition(parser, key.Node())
	for !key.IsLast() {
		key.Next()
	}
	raw := key.Node().Raw
	data := parser.Data()
	start := len(data) - len(bytes.TrimLeft(data[raw.Offset+raw.Length:], " =\t"))
	value := expression.Value()
	end := recordValuePositions(parser, value, path, start, positions)
	at := rangePosition(parser, parser.Range(data[start:end]))
	position.value = &at
	if value.Kind == unstable.String {
		quote := stringDelimiterWidth(data[start:end])
		if end-start > 2*quote {
			content := rangePosition(parser, parser.Range(data[start+quote:end-quote]))
			position.content = &content
		}
	}
	position.brace = positions[path].brace
	positions[path] = position
	return end
}

// Container nodes expose only their opening delimiter or no raw range at all.
func recordValuePositions(parser *unstable.Parser, node *unstable.Node, path string, start int, positions sourcePositions) int {
	data := parser.Data()
	switch node.Kind {
	case unstable.Array:
		end := start + 1
		index := 0
		for child := node.Child(); child != nil; child = child.Next() {
			childPath := indexPath(path, index)
			first := skipContainerTrivia(data, end)
			end = recordValuePositions(parser, child, childPath, first, positions)
			positions[childPath] = rangePosition(parser, parser.Range(data[first:end]))
			index++
		}
		return skipContainerTrivia(data, end) + 1
	case unstable.InlineTable:
		end := start + 1
		for child := node.Child(); child != nil; child = child.Next() {
			key := child.Key()
			if key.Next() {
				end = recordKeyValuePositions(parser, child, path, key, positions)
			}
		}
		return skipContainerTrivia(data, end) + 1
	case unstable.String:
		if strings.HasSuffix(path, ".def") {
			recordDefinitionPositions(parser, node, path, positions)
		}
	case unstable.Invalid, unstable.Comment, unstable.Key, unstable.Table, unstable.ArrayTable,
		unstable.KeyValue, unstable.Bool, unstable.Float, unstable.Integer,
		unstable.LocalDate, unstable.LocalTime, unstable.LocalDateTime, unstable.DateTime:
	}
	return int(node.Raw.Offset + node.Raw.Length)
}

func skipContainerTrivia(data []byte, offset int) int {
	for offset < len(data) {
		switch data[offset] {
		case ' ', '\t', '\r', '\n', ',':
			offset++
		case '#':
			newline := bytes.IndexByte(data[offset:], '\n')
			if newline < 0 {
				return len(data)
			}
			offset += newline + 1
		default:
			return offset
		}
	}
	return offset
}

func keyPath(node *unstable.Node) string {
	var parts []string
	for key := node.Key(); key.Next(); {
		parts = append(parts, pathSegment(string(key.Node().Data)))
	}
	return strings.Join(parts, ".")
}

func nodePosition(parser *unstable.Parser, node *unstable.Node) sourcePosition {
	return rangePosition(parser, node.Raw)
}

func rangePosition(parser *unstable.Parser, span unstable.Range) sourcePosition {
	shape := parser.Shape(span)
	return sourcePosition{line: shape.Start.Line, column: shape.Start.Column, length: int(span.Length)}
}

func appendPath(parent, child string) string {
	if parent == "" {
		return child
	}
	return parent + "." + child
}

func syntaxError(data []byte, path string, err error) *SyntaxError {
	at := Location{Path: path, Line: 1, Column: 1, Length: 1}
	if decodeError, ok := errors.AsType[*toml.DecodeError](err); ok {
		at.Line, at.Column = decodeError.Position()
	}
	at.Text = lineText(splitSourceLines(data), at.Line)
	return &SyntaxError{at: at, cause: err, message: strings.TrimPrefix(err.Error(), "toml: ")}
}

func lineText(lines []string, line int) string {
	if line < 1 || line > len(lines) {
		return ""
	}
	return lines[line-1]
}

func splitSourceLines(data []byte) []string {
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	for i := range lines {
		lines[i] = strings.TrimSuffix(lines[i], "\r")
	}
	return lines
}

func joinPath(parent, key string) string {
	return appendPath(parent, pathSegment(key))
}

func pathSegment(key string) string {
	if key == "" || strings.ContainsAny(key, ".[]\"") {
		return strconv.Quote(key)
	}
	return key
}

func (p *validator) reportf(path, format string, args ...any) {
	positions := p.positions
	position := positions[path]
	for i := len(path) - 1; position.line == 0 && i >= 0; i-- {
		if path[i] == '.' || path[i] == '[' {
			position = positions[path[:i]]
		}
	}
	if position.line == 0 {
		position.line, position.column = 1, 1
	}
	p.addFieldError(path, position, fmt.Sprintf(format, args...))
}

func (p *validator) reportValuef(path, format string, args ...any) {
	if value := p.positions[path].value; value != nil {
		p.addFieldError(path, *value, fmt.Sprintf(format, args...))
		return
	}
	p.reportf(path, format, args...)
}

func (p *validator) reportContentf(path, format string, args ...any) {
	if content := p.positions[path].content; content != nil {
		p.addFieldError(path, *content, fmt.Sprintf(format, args...))
		return
	}
	p.reportValuef(path, format, args...)
}

func (p *validator) addFieldError(path string, position sourcePosition, message string) {
	p.issues = append(p.issues, &FieldError{
		at:      Location{Path: p.path, Line: position.line, Column: position.column, Length: position.length, Text: lineText(p.lines, position.line)},
		field:   path,
		message: message,
	})
}
