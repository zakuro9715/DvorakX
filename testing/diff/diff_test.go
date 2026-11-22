package diff

import (
	"reflect"
	"strings"
	"testing"

	fatihcolor "github.com/fatih/color"
)

func TestTokensAlignsAndMarksDifferences(t *testing.T) {
	got := Tokens("a x c", "a longer c")
	want := []Column{
		{Want: "a", Got: "a"},
		{Want: "x", Got: "longer", Different: true},
		{Want: "c", Got: "c"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %#v, want %#v", got, want)
	}
}

func TestTokensAlignsInsertions(t *testing.T) {
	got := Tokens("a", "a b")
	want := []Column{
		{Want: "a", Got: "a"},
		{Got: "b", Different: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %#v, want %#v", got, want)
	}
}

func TestTokensPairsReplacementRuns(t *testing.T) {
	got := Tokens("0x41 0x41 0x43", "0x42 0x42 0x43")
	want := []Column{
		{Want: "0x41", Got: "0x42", Different: true},
		{Want: "0x41", Got: "0x42", Different: true},
		{Want: "0x43", Got: "0x43"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %#v, want %#v", got, want)
	}
}

func TestTokensPrefersReplacementOverSharedTokenAlignment(t *testing.T) {
	got := Tokens("0x41 0x42 0x43", "0x42 0x42 0x43")
	want := []Column{
		{Want: "0x41", Got: "0x42", Different: true},
		{Want: "0x42", Got: "0x42"},
		{Want: "0x43", Got: "0x43"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("columns = %#v, want %#v", got, want)
	}
}

func TestFormatColorsAndAlignsTokens(t *testing.T) {
	previousColor := fatihcolor.NoColor
	fatihcolor.NoColor = true
	t.Cleanup(func() {
		fatihcolor.NoColor = previousColor
	})

	want, got := Format("a x c", "a longer c")
	if want != "\x1b[38;2;255;95;135ma \x1b[7mx\x1b[27m"+strings.Repeat(" ", 6)+"c\x1b[0;22;0;0;0m" {
		t.Errorf("want = %q", want)
	}

	if got != "\x1b[38;2;95;215;135ma \x1b[7mlonger\x1b[27m c\x1b[0;22;0;0;0m" {
		t.Errorf("got = %q", got)
	}
}

func TestFormatterDisableColor(t *testing.T) {
	formatter := NewFormatter().DisableColor()
	want, got := formatter.Format("a x c", "a longer c")
	if want != "a x      c" {
		t.Errorf("want = %q", want)
	}
	if got != "a longer c" {
		t.Errorf("got = %q", got)
	}
}
