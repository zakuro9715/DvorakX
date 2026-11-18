package keyboard

import (
	"errors"
	"reflect"
	"testing"
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
		NewMappingsWithActions(nil, map[uint16]bool{0x41: true}), // A
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
