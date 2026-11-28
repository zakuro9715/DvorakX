package parser

import (
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/pelletier/go-toml/v2/unstable"
)

func recordDefinitionPositions(parser *unstable.Parser, node *unstable.Node, path string, positions sourcePositions) {
	tokens, err := splitDefinition(string(node.Data))
	spans := definitionSourceSpans(parser.Raw(node.Raw))
	if len(spans) != len(node.Data) {
		return
	}
	if err != nil {
		if problem, ok := errors.AsType[*definitionError](err); ok {
			at := decodedSpanPosition(parser, node, spans, problem.start, problem.end)
			positions[path] = sourcePosition{brace: &at}
		}
		return
	}
	for i, token := range tokens {
		positions[indexPath(path, i)] = decodedSpanPosition(parser, node, spans, token.start, token.end)
	}
}

func decodedSpanPosition(parser *unstable.Parser, node *unstable.Node, spans []byteSpan, first, last int) sourcePosition {
	start := spans[first].start
	end := spans[last-1].end
	return rangePosition(parser, unstable.Range{Offset: node.Raw.Offset + uint32(start), Length: uint32(end - start)})
}

type byteSpan struct {
	start int
	end   int
}

// definitionSourceSpans maps each decoded byte to the original TOML spelling.
func definitionSourceSpans(raw []byte) []byteSpan {
	quotes := stringDelimiterWidth(raw)
	end := len(raw) - quotes
	i := quotes
	if quotes == 3 && i < end {
		if raw[i] == '\r' {
			i++
		}
		if i < end && raw[i] == '\n' {
			i++
		}
	}
	spans := make([]byteSpan, 0, end-i)
	for i < end {
		start := i
		size := 1
		if raw[0] == '"' && raw[i] == '\\' {
			i, size = definitionEscapeSpan(raw, i+1, end)
			if size == 0 {
				continue
			}
		} else {
			i++
		}
		for range size {
			spans = append(spans, byteSpan{start: start, end: i})
		}
	}
	return spans
}

func stringDelimiterWidth(raw []byte) int {
	if len(raw) >= 6 && raw[1] == raw[0] && raw[2] == raw[0] {
		return 3
	}
	return 1
}

// The TOML decoder has already validated the escape sequence.
func definitionEscapeSpan(raw []byte, i, end int) (next, size int) {
	if tomlStringWhitespace(raw[i]) {
		for i < end && tomlStringWhitespace(raw[i]) {
			i++
		}
		return i, 0
	}
	digits := map[byte]int{'x': 2, 'u': 4, 'U': 8}[raw[i]]
	i++
	if digits == 0 {
		return i, 1
	}
	number, _ := strconv.ParseUint(string(raw[i:i+digits]), 16, 32)
	return i + digits, utf8.RuneLen(rune(number))
}

func tomlStringWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}
