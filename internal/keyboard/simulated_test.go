package keyboard

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zakuro9715/DvorakX/internal/config"
)

func TestSimulatedKeyboardRemapsPressedKeyAfterMappingsReload(t *testing.T) {
	t.Parallel()

	mappings := NewMappings(map[uint16]uint16{0x41: 0x42}) // A -> B
	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(mappings, output)

	// A pressed key keeps its original destination even after mappings reload.
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41}); !handled || err != nil { // A
		t.Fatalf("handle key down = (%t, %v), want (true, nil)", handled, err)
	}
	mappings.Replace(map[uint16]uint16{0x41: 0x43}) // A -> C
	// The release must use the destination selected at key-down time.
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true}); !handled || err != nil { // A
		t.Fatalf("handle key up = (%t, %v), want (true, nil)", handled, err)
	}

	want := []Event{{VirtualKey: 0x42}, {VirtualKey: 0x42, KeyUp: true}} // B
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesUnmappedEventsThrough(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output) // A -> B

	// An unmapped key is left for the operating system and emits no replacement.
	handled, err := keyboard.Handle(Event{VirtualKey: 0x43}) // C
	if handled || err != nil {
		t.Fatalf("handle unmapped key = (%t, %v), want (false, nil)", handled, err)
	}
	if len(output.events) != 0 {
		t.Errorf("output events = %#v, want none", output.events)
	}
}

func TestSimulatedKeyboardSuppressesNoopEvents(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(
		NewMappingsWithActions(nil, map[uint16]bool{0x41: true}, nil), // A
		output,
	)

	// A noop consumes both events without sending anything to the output.
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41}); !handled || err != nil { // A
		t.Fatalf("handle noop key down = (%t, %v), want (true, nil)", handled, err)
	}
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true}); !handled || err != nil { // A
		t.Fatalf("handle noop key up = (%t, %v), want (true, nil)", handled, err)
	}
	if len(output.events) != 0 {
		t.Errorf("output events = %#v, want none", output.events)
	}
}

func TestSimulatedKeyboardSendsChordActionAndConsumesTrigger(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(
		NewMappingsWithActions(nil, nil, map[uint16][]uint16{
			0x41: {0x11, 0x43}, // A triggers Ctrl+C.
		}),
		output,
	)

	for _, event := range []Event{
		{VirtualKey: 0x41},              // A
		{VirtualKey: 0x41, KeyUp: true}, // A
	} {
		if handled, err := keyboard.Handle(event); !handled || err != nil {
			t.Fatalf("handle %v = (%t, %v), want (true, nil)", event, handled, err)
		}
	}

	want := []Event{
		{VirtualKey: 0x11},              // Ctrl down
		{VirtualKey: 0x43},              // C down
		{VirtualKey: 0x43, KeyUp: true}, // C up
		{VirtualKey: 0x11, KeyUp: true}, // Ctrl up
	}
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesThroughWhenSendKeysFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	keyboard := NewSimulatedKeyboard(
		NewMappingsWithActions(nil, nil, map[uint16][]uint16{0x41: {0x11}}), // A triggers Ctrl.
		output,
	)

	handled, err := keyboard.Handle(Event{VirtualKey: 0x41}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle action = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true}); handled || err != nil {
		t.Fatalf("handle failed action release = (%t, %v), want (false, nil)", handled, err)
	}
}

func TestSimulatedKeyboardPassesThroughWhenOutputFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output) // A -> B

	// An output failure makes the event unhandled while preserving the error.
	handled, err := keyboard.Handle(Event{VirtualKey: 0x41}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle key = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if want := []Event{{VirtualKey: 0x42}}; !reflect.DeepEqual(output.events, want) { // B
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesThroughReleaseWithoutPress(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output) // A -> B

	// A release without a tracked press must pass through unchanged.
	handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true}) // A
	if handled || err != nil {
		t.Fatalf("handle key up = (%t, %v), want (false, nil)", handled, err)
	}
	if len(output.events) != 0 {
		t.Errorf("output events = %#v, want none", output.events)
	}
}

func TestSimulatedKeyboardAppliesDefinitionsInOrder(t *testing.T) {
	loaded, err := config.Parse([]byte(`
[global]
remaps = { "a" = "b" }

[meta]
default = "&layer"

[src]
def = ["a", "b", "c"]

[layer]
def = ["e", "f", "g"]
remaps = { f = "x" }
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappingsFromStages(loaded.Stages), output)
	for _, key := range []uint16{0x41, 0x42} { // A and B
		if handled, err := keyboard.Handle(Event{VirtualKey: key}); !handled || err != nil {
			t.Fatalf("handle key down 0x%X = (%t, %v), want (true, nil)", key, handled, err)
		}
		if handled, err := keyboard.Handle(Event{VirtualKey: key, KeyUp: true}); !handled || err != nil {
			t.Fatalf("handle key up 0x%X = (%t, %v), want (true, nil)", key, handled, err)
		}
	}

	want := []Event{
		{VirtualKey: 0x58}, {VirtualKey: 0x58, KeyUp: true}, // X
		{VirtualKey: 0x58}, {VirtualKey: 0x58, KeyUp: true}, // X
	}
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

type recordingOutput struct {
	events []Event
	err    error
}

func newRecordingOutput() *recordingOutput {
	return new(recordingOutput)
}

func (o *recordingOutput) Send(event Event) error {
	o.events = append(o.events, event)
	return o.err
}
