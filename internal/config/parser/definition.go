package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type definitionToken struct {
	value   string
	escaped bool
	start   int
	end     int
}

func unescapeKeyName(name string) (string, error) {
	if !strings.ContainsAny(name, "{}") {
		return name, nil
	}
	if len(name) < 3 || name[0] != '{' || name[len(name)-1] != '}' {
		return "", fmt.Errorf("%q must be enclosed by one pair of braces", name)
	}
	unescaped := name[1 : len(name)-1]
	if unescaped == "" || strings.ContainsAny(unescaped, "{}") {
		return "", fmt.Errorf("%q contains invalid braces", name)
	}
	return unescaped, nil
}

func definitionText(definition string) (string, []int) {
	var text strings.Builder
	var offsets []int
	offset := 0
	for line := range strings.SplitSeq(definition, "\n") {
		trimmedLeft := strings.TrimLeftFunc(line, unicode.IsSpace)
		trimmed := strings.TrimRightFunc(trimmedLeft, unicode.IsSpace)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			if text.Len() > 0 {
				text.WriteByte(' ')
				offsets = append(offsets, offset-1)
			}
			start := offset + len(line) - len(trimmedLeft)
			text.WriteString(trimmed)
			for i := range len(trimmed) {
				offsets = append(offsets, start+i)
			}
		}
		offset += len(line) + 1
	}
	return text.String(), offsets
}

type definitionError struct {
	message    string
	start, end int
}

func (e *definitionError) Error() string { return e.message }

func splitDefinition(definition string) ([]definitionToken, error) {
	text, offsets := definitionText(definition)
	var tokens []definitionToken
	var token strings.Builder
	escaped := false
	start := 0
	emit := func(end int) {
		tokens = append(tokens, definitionToken{value: token.String(), escaped: escaped, start: offsets[start], end: offsets[end-1] + 1})
		token.Reset()
	}
	fail := func(message string, first, last int) ([]definitionToken, error) {
		return nil, &definitionError{message: message, start: offsets[first], end: offsets[last-1] + 1}
	}
	for i, character := range text {
		switch {
		case escaped:
			switch character {
			case '{':
				return fail("nested { is not allowed", i, i+1)
			case '}':
				if token.Len() == 0 {
					return fail("{} is not allowed", start, i+1)
				}
				emit(i + utf8.RuneLen(character))
				escaped = false
			default:
				token.WriteRune(character)
			}
		case character == '{':
			if token.Len() != 0 {
				return fail("{ must start a key definition", i, i+1)
			}
			start = i
			escaped = true
		case character == '}':
			return fail("unexpected }", i, i+1)
		case unicode.IsSpace(character) || character == ',' || character == '|':
			if token.Len() != 0 {
				emit(i)
			}
		default:
			if token.Len() == 0 {
				start = i
			}
			token.WriteRune(character)
		}
	}
	if escaped {
		return fail("unclosed {", start, len(text))
	}
	if token.Len() != 0 {
		emit(len(text))
	}
	return tokens, nil
}

func (p *validator) literal(value, path string) (definitionToken, bool) {
	name, err := unescapeKeyName(value)
	if err != nil {
		p.reportf(path, "%v", err)
		return definitionToken{}, false
	}
	return definitionToken{value: name, escaped: name != value}, true
}

func (p *validator) tokens(value any, path string) []definitionToken {
	switch value := value.(type) {
	case string:
		tokens, err := splitDefinition(value)
		if err != nil {
			if at := p.positions[path].brace; at != nil {
				p.addFieldError(path, *at, err.Error())
			} else {
				p.reportValuef(path, "%v", err)
			}
			return nil
		}
		if len(tokens) == 0 {
			p.reportValuef(path, "%s must contain at least one key", path)
			return nil
		}
		return tokens
	case []any:
		if len(value) == 0 {
			p.reportValuef(path, "%s must contain at least one key", path)
			return nil
		}
		tokens := make([]definitionToken, len(value))
		valid := true
		for i, entry := range value {
			item := indexPath(path, i)
			name, ok := entry.(string)
			if !ok {
				p.reportf(item, "%s must be a string", item)
				valid = false
				continue
			}
			token, ok := p.literal(name, item)
			valid = valid && ok
			tokens[i] = token
		}
		if valid {
			return tokens
		}
	case nil:
		p.reportf(path, "[%s].def is required", strings.TrimSuffix(path, ".def"))
	default:
		p.reportValuef(path, "%s must be a string or string array", path)
	}
	return nil
}
