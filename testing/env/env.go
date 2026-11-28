// Package env provides helpers for testing keyboard configurations.
//
// Create an Environment with New, configure it with Config or LoadConfig,
// and use the event DSL with Simulate, Expect, or Test.
package env

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/zakuro9715/DvorakZ/internal/config"
	"github.com/zakuro9715/DvorakZ/internal/config/parser"
	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
	"github.com/zakuro9715/DvorakZ/testing/diff"
)

// Environment applies keyboard events to a simulated keyboard and records its output.
type Environment struct {
	t           testingT
	output      *recordingOutput
	keyboard    *keyboard.SimulatedKeyboard
	formatter   *diff.Formatter
	inputEvents []keyboard.Event
	inputTokens []string
}

type testingT interface {
	Helper()
	Fatalf(string, ...any)
	Errorf(string, ...any)
}

// New returns a new testing environment with an empty configuration.
func New(t testingT) *Environment {
	t.Helper()
	output := new(recordingOutput)
	environment := &Environment{
		t:         t,
		output:    output,
		formatter: diff.NewFormatter(),
		keyboard: keyboard.NewSimulatedKeyboard(
			keyboard.NewMappings(nil),
			output,
		),
	}
	return environment
}

// DisableColor disables ANSI colors in diff output for this environment.
func (environment *Environment) DisableColor() *Environment {
	environment.t.Helper()
	environment.formatter.DisableColor()
	return environment
}

// Config configures the environment from TOML text.
func Config(t testingT, text string) *Environment {
	return New(t).Config(text)
}

// Config configures the environment from TOML text.
func (environment *Environment) Config(text string) *Environment {
	environment.t.Helper()
	loaded, err := parser.Parse([]byte(text))
	if err != nil {
		environment.t.Fatalf("parse test configuration: %v", err)
	}
	environment.replaceConfig(loaded)
	return environment
}

// LoadConfig configures the environment from a file relative to the calling test source.
func LoadConfig(t testingT, path string) *Environment {
	return New(t).loadConfig(path, 2)
}

// LoadConfig configures the environment from a file relative to the calling test source.
func (environment *Environment) LoadConfig(path string) *Environment {
	return environment.loadConfig(path, 2)
}

func (environment *Environment) loadConfig(path string, callerSkip int) *Environment {
	environment.t.Helper()
	_, file, _, ok := runtime.Caller(callerSkip)
	if !ok {
		environment.t.Fatalf("locate test source for config %q", path)
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(file), path)
	}
	path = filepath.ToSlash(path)
	loaded, err := parser.LoadFile(path)
	if err != nil {
		environment.t.Fatalf("load test configuration %q: %v", path, err)
	}
	environment.replaceConfig(loaded)
	return environment
}

// Simulate sends input events described by the DSL to the simulated keyboard.
func (environment *Environment) Simulate(text string) *Environment {
	environment.t.Helper()
	events := parseEvents(environment.t, text)
	environment.inputEvents = events
	environment.inputTokens = strings.Fields(stripComments(text))
	for _, event := range events {
		handled, err := environment.keyboard.Handle(event)
		if err != nil {
			environment.t.Fatalf("simulate event %#v: %v", event, err)
		}
		if !handled {
			if err := environment.output.Send(event); err != nil {
				environment.t.Fatalf("record passthrough event %#v: %v", event, err)
			}
		}
	}
	return environment
}

// Test simulates input events and checks the recorded output.
func (environment *Environment) Test(input, expected string) *Environment {
	environment.t.Helper()
	return environment.Simulate(input).expect(expected)
}

// Expect checks that the recorded output matches events described by the DSL.
func (environment *Environment) Expect(text string) {
	environment.t.Helper()
	environment.expect(text)
}

func (environment *Environment) expect(text string) *Environment {
	want := parseEvents(environment.t, text)
	if !reflect.DeepEqual(environment.output.events, want) {
		wantText, gotText := environment.formatter.Format(
			stripComments(text),
			formatEvents(environment.output.events, environment.inputEvents, environment.inputTokens),
		)
		environment.t.Errorf(
			"output events mismatch:\n- %s\n+ %s",
			wantText,
			gotText,
		)
	}

	return environment
}

func (environment *Environment) replaceConfig(loaded config.Config) {
	environment.t.Helper()
	mappings := keyboard.NewMappingsFromConfig(loaded)
	environment.keyboard = keyboard.NewSimulatedKeyboard(mappings, environment.output)
	environment.output.events = nil
}

type recordingOutput struct {
	events []keyboard.Event
}

func (output *recordingOutput) Send(event keyboard.Event) error {
	output.events = append(output.events, event)
	return nil
}

func formatEvents(events, sourceEvents []keyboard.Event, sourceTokens []string) string {
	tokens := make([]string, 0, len(events))
	for index := 0; index < len(events); index++ {
		sourceIndex := 0
		sourceOffset := 0
		for sourceOffset < index && sourceOffset < len(sourceEvents) {
			sourceOffset += eventTokenLength(sourceEvents, sourceOffset)
			sourceIndex++
		}
		if sourceOffset == index && sourceIndex < len(sourceTokens) {
			length := eventTokenLength(sourceEvents, sourceOffset)
			if index+length <= len(events) &&
				reflect.DeepEqual(events[index:index+length], sourceEvents[sourceOffset:sourceOffset+length]) {
				tokens = append(tokens, sourceTokens[sourceIndex])
				index += length - 1
				continue
			}
		}

		event := events[index]
		key := formatKey(event.Key)
		length := eventTokenLength(events, index)
		switch {
		case length == 2:
			tokens = append(tokens, key)
			index += length - 1
		case event.KeyUp:
			tokens = append(tokens, key+":up")
		default:
			tokens = append(tokens, key+":down")
		}
	}
	return strings.Join(tokens, " ")
}

func eventTokenLength(events []keyboard.Event, index int) int {
	if index+1 < len(events) &&
		!events[index].KeyUp &&
		events[index+1].Key == events[index].Key &&
		events[index+1].KeyUp {
		return 2
	}
	return 1
}

func formatKey(key keys.Key) string {
	if name := keys.Name(key); name != "" {
		return name
	}
	if key.Extended {
		return fmt.Sprintf("0xE0%02X", key.ScanCode)
	}
	switch key.ScanCode {
	case 0x1D:
		return "ctrl"
	case 0x38:
		return "alt"
	case 0x2A, 0x36:
		return "shift"
	case 0x3A:
		return "caps"
	case 0x01:
		return "esc"
	case 0x0F:
		return "tab"
	case 0x1C:
		return "enter"
	case 0x39:
		return "space"
	default:
		return fmt.Sprintf("0x%X", key.ScanCode)
	}
}
