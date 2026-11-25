package keyboard_test

import (
	"errors"
	"testing"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
	"github.com/zakuro9715/DvorakZ/testing/env"
)

func TestSimulatedKeyboardRemapsPressedKeyAfterMappingsReload(t *testing.T) {
	t.Parallel()

	mappings := keyboard.NewMappings(map[keys.Key]keys.Key{key(0x1E): key(0x30)}) // A -> B
	output := new(recordingOutput)
	simulated := keyboard.NewSimulatedKeyboard(mappings, output)

	// A pressed key keeps its original destination after the mapping reload.
	if handled, err := simulated.Handle(keyboard.Event{Key: key(0x1E)}); !handled || err != nil { // A
		t.Fatalf("handle key down = (%t, %v), want (true, nil)", handled, err)
	}
	mappings.Replace(map[keys.Key]keys.Key{key(0x1E): key(0x2E)})                                              // A -> C
	if handled, err := simulated.Handle(keyboard.Event{Key: key(0x1E), KeyUp: true}); !handled || err != nil { // A
		t.Fatalf("handle key up = (%t, %v), want (true, nil)", handled, err)
	}

	if got, want := output.events, []keyboard.Event{
		{Key: key(0x30)}, {Key: key(0x30), KeyUp: true}, // B
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
		keyboard.NewMappingsWithActions(nil, nil, map[keys.Key][]keys.Key{key(0x1E): {key(0x1D)}}), // A triggers Ctrl.
		output,
	)

	handled, err := simulated.Handle(keyboard.Event{Key: key(0x1E)}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle action = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if handled, err := simulated.Handle(keyboard.Event{Key: key(0x1E), KeyUp: true}); handled || err != nil {
		t.Fatalf("handle failed action release = (%t, %v), want (false, nil)", handled, err)
	}
}

func TestSimulatedKeyboardPassesThroughWhenOutputFails(t *testing.T) {
	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	simulated := keyboard.NewSimulatedKeyboard(
		keyboard.NewMappings(map[keys.Key]keys.Key{key(0x1E): key(0x30)}), // A -> B
		output,
	)

	// An output failure makes the event unhandled while preserving the error.
	handled, err := simulated.Handle(keyboard.Event{Key: key(0x1E)}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle key = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if len(output.events) != 1 || output.events[0] != (keyboard.Event{Key: key(0x30)}) { // B
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

func key(scanCode uint16) keys.Key {
	return keys.Key{ScanCode: scanCode}
}

func (output *recordingOutput) Send(event keyboard.Event) error {
	output.events = append(output.events, event)
	return output.err
}
