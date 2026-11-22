// Package diff aligns space-separated tokens and identifies differences.
package diff

import (
	"strings"
	"unicode/utf8"

	fatihcolor "github.com/fatih/color"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Column contains the tokens displayed at the same position in both outputs.
type Column struct {
	Want      string
	Got       string
	Different bool
}

// Tokens returns aligned columns for the expected and actual token sequences.
func Tokens(want, got string) []Column {
	wantTokens := strings.Fields(want)
	gotTokens := strings.Fields(got)
	ids := make(map[string]rune, len(wantTokens)+len(gotTokens))
	tokens := make(map[rune]string, len(wantTokens)+len(gotTokens))
	nextID := rune(0x100000)
	encode := func(token string) rune {
		if id, ok := ids[token]; ok {
			return id
		}
		id := nextID
		nextID++
		ids[token] = id
		tokens[id] = token
		return id
	}
	wantIDs := make([]rune, len(wantTokens))
	for index, token := range wantTokens {
		wantIDs[index] = encode(token)
	}
	gotIDs := make([]rune, len(gotTokens))
	for index, token := range gotTokens {
		gotIDs[index] = encode(token)
	}

	diffs := diffmatchpatch.New().DiffMainRunes(wantIDs, gotIDs, false)
	columns := make([]Column, 0, len(wantTokens)+len(gotTokens))
	for index := 0; index < len(diffs); {
		if diffs[index].Type == diffmatchpatch.DiffEqual {
			for _, id := range []rune(diffs[index].Text) {
				columns = append(columns, Column{Want: tokens[id], Got: tokens[id]})
			}
			index++
			continue
		}

		wantRun := make([]string, 0)
		gotRun := make([]string, 0)
		for index < len(diffs) && diffs[index].Type != diffmatchpatch.DiffEqual {
			for _, id := range []rune(diffs[index].Text) {
				if diffs[index].Type == diffmatchpatch.DiffDelete {
					wantRun = append(wantRun, tokens[id])
				} else {
					gotRun = append(gotRun, tokens[id])
				}
			}
			index++
		}
		for pairIndex := 0; pairIndex < max(len(wantRun), len(gotRun)); pairIndex++ {
			column := Column{Different: true}
			if pairIndex < len(wantRun) {
				column.Want = wantRun[pairIndex]
			}
			if pairIndex < len(gotRun) {
				column.Got = gotRun[pairIndex]
			}
			columns = append(columns, column)
		}
	}
	return columns
}

// Format returns aligned, colored expected and actual diff lines.
func Format(want, got string) (string, string) {
	return NewFormatter().Format(want, got)
}

// Formatter renders aligned diff output with configurable colors.
type Formatter struct {
	wantColor       *fatihcolor.Color
	gotColor        *fatihcolor.Color
	backgroundColor *fatihcolor.Color
	enabled         bool
}

// NewFormatter returns a formatter with colors enabled.
func NewFormatter() *Formatter {
	return &Formatter{
		wantColor:       fatihcolor.RGB(255, 95, 135),
		gotColor:        fatihcolor.RGB(95, 215, 135),
		backgroundColor: fatihcolor.New(fatihcolor.ReverseVideo),
		enabled:         true,
	}
}

// DisableColor disables colors for this formatter.
func (formatter *Formatter) DisableColor() *Formatter {
	formatter.wantColor.DisableColor()
	formatter.gotColor.DisableColor()
	formatter.backgroundColor.DisableColor()
	formatter.enabled = false
	return formatter
}

// Format returns aligned expected and actual diff lines.
func (formatter *Formatter) Format(want, got string) (string, string) {
	columns := Tokens(want, got)
	if formatter.enabled {
		formatter.wantColor.EnableColor()
		formatter.gotColor.EnableColor()
		formatter.backgroundColor.EnableColor()
	} else {
		formatter.wantColor.DisableColor()
		formatter.gotColor.DisableColor()
		formatter.backgroundColor.DisableColor()
	}
	return formatter.wantColor.Sprint(render(columns, true, formatter.backgroundColor)),
		formatter.gotColor.Sprint(render(columns, false, formatter.backgroundColor))
}

func render(columns []Column, want bool, backgroundColor *fatihcolor.Color) string {
	var text strings.Builder
	lastIndex := -1
	for index, column := range columns {
		if (want && column.Want != "") || (!want && column.Got != "") {
			lastIndex = index
		}
	}
	for index, column := range columns {
		value := column.Got
		if want {
			value = column.Want
		}
		width := max(utf8.RuneCountInString(column.Want), utf8.RuneCountInString(column.Got))
		if column.Different && value != "" {
			text.WriteString(backgroundColor.Sprint(value))
		} else {
			text.WriteString(value)
		}
		if index < lastIndex {
			for padding := utf8.RuneCountInString(value); padding < width; padding++ {
				text.WriteByte(' ')
			}
			text.WriteByte(' ')
		}
	}
	return text.String()
}
