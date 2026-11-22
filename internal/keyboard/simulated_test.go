package keyboard_test

import (
	"errors"
	"testing"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
	"github.com/zakuro9715/DvorakX/testing/env"
)

func TestSimulatedKeyboardRemapsPressedKeyAfterMappingsReload(t *testing.T) {
	t.Parallel()

	mappings := keyboard.NewMappings(map[uint16]uint16{0x41: 0x42}) // A -> B
	output := new(recordingOutput)
	simulated := keyboard.NewSimulatedKeyboard(mappings, output)

	// A pressed key keeps its original destination after the mapping reload.
	if handled, err := simulated.Handle(keyboard.Event{VirtualKey: 0x41}); !handled || err != nil { // A
		t.Fatalf("handle key down = (%t, %v), want (true, nil)", handled, err)
	}
	mappings.Replace(map[uint16]uint16{0x41: 0x43})                                                              // A -> C
	if handled, err := simulated.Handle(keyboard.Event{VirtualKey: 0x41, KeyUp: true}); !handled || err != nil { // A
		t.Fatalf("handle key up = (%t, %v), want (true, nil)", handled, err)
	}

	if got, want := output.events, []keyboard.Event{
		{VirtualKey: 0x42}, {VirtualKey: 0x42, KeyUp: true}, // B
	}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("output events = %#v, want %#v", got, want)
	}
}

func TestSimulatedKeyboardPassesUnmappedEventsThrough(t *testing.T) {
	t.Parallel()

	env.New(t).
		Config(`[global]
remaps = { a = "b" }
`).
		Test("c", "c")
}

func TestSimulatedKeyboardSuppressesNoopEvents(t *testing.T) {
	t.Parallel()

	env.New(t).
		Config(`[global]
remaps = { a = "@disabled" }

[global.actions]
disabled = { noop = [] }
`).
		Test("a", "")
}

func TestSimulatedKeyboardSendsChordActionAndConsumesTrigger(t *testing.T) {
	t.Parallel()

	env.New(t).
		Config(`[global]
remaps = { a = "@copy" }

[global.actions]
copy = { send-keys = ["ctrl", "c"] }
`).
		Test("a", "ctrl:down c ctrl:up")
}

func TestSimulatedKeyboardPassesThroughWhenSendKeysFails(t *testing.T) {
	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	simulated := keyboard.NewSimulatedKeyboard(
		keyboard.NewMappingsWithActions(nil, nil, map[uint16][]uint16{0x41: {0x11}}), // A triggers Ctrl.
		output,
	)

	handled, err := simulated.Handle(keyboard.Event{VirtualKey: 0x41}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle action = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if handled, err := simulated.Handle(keyboard.Event{VirtualKey: 0x41, KeyUp: true}); handled || err != nil {
		t.Fatalf("handle failed action release = (%t, %v), want (false, nil)", handled, err)
	}
}

func TestSimulatedKeyboardPassesThroughWhenOutputFails(t *testing.T) {
	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	simulated := keyboard.NewSimulatedKeyboard(
		keyboard.NewMappings(map[uint16]uint16{0x41: 0x42}), // A -> B
		output,
	)

	// An output failure makes the event unhandled while preserving the error.
	handled, err := simulated.Handle(keyboard.Event{VirtualKey: 0x41}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle key = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if len(output.events) != 1 || output.events[0] != (keyboard.Event{VirtualKey: 0x42}) { // B
		t.Errorf("output events = %#v, want B down", output.events)
	}
}

func TestSimulatedKeyboardPassesThroughReleaseWithoutPress(t *testing.T) {
	t.Parallel()

	env.New(t).
		Config(`[global]
remaps = { a = "b" }
`).
		Test("a:up", "a:up")
}

func TestSimulatedKeyboardAppliesDefinitionsInOrder(t *testing.T) {
	t.Parallel()

	env.New(t).
		Config(`
[global]
remaps = { "a" = "b" }

[meta]
default = "&layer"

[src]
def = ["a", "b", "c"]

[layer]
def = ["e", "f", "g"]
remaps = { f = "x" }
`).
		Test("a b", "x x")
}

type recordingOutput struct {
	events []keyboard.Event
	err    error
}

func (output *recordingOutput) Send(event keyboard.Event) error {
	output.events = append(output.events, event)
	return output.err
}
