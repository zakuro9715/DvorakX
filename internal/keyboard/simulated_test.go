package keyboard

import (
	"errors"
	"reflect"
	"testing"
)

func TestSimulatedKeyboardRemapsPressedKeyAfterMappingsReload(t *testing.T) {
	t.Parallel()

	mappings := NewMappings(map[uint16]uint16{0x1E: 0x30}) // A -> B
	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(mappings, output)

	// A pressed key keeps its original destination even after mappings reload.
	if handled, err := keyboard.Handle(Event{ScanCode: 0x1E}); !handled || err != nil { // A
		t.Fatalf("handle key down = (%t, %v), want (true, nil)", handled, err)
	}
	mappings.Replace(map[uint16]uint16{0x1E: 0x2E}) // A -> C
	// The release must use the destination selected at key-down time.
	if handled, err := keyboard.Handle(Event{ScanCode: 0x1E, KeyUp: true}); !handled || err != nil { // A
		t.Fatalf("handle key up = (%t, %v), want (true, nil)", handled, err)
	}

	want := []Event{{ScanCode: 0x30}, {ScanCode: 0x30, KeyUp: true}} // B
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesUnmappedEventsThrough(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x1E: 0x30}), output) // A -> B

	// An unmapped key is left for the operating system and emits no replacement.
	handled, err := keyboard.Handle(Event{ScanCode: 0x2E}) // C
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
		NewMappingsWithActions(nil, map[uint16]bool{0x1E: true}), // A
		output,
	)

	// A noop consumes both events without sending anything to the output.
	if handled, err := keyboard.Handle(Event{ScanCode: 0x1E}); !handled || err != nil { // A
		t.Fatalf("handle noop key down = (%t, %v), want (true, nil)", handled, err)
	}
	if handled, err := keyboard.Handle(Event{ScanCode: 0x1E, KeyUp: true}); !handled || err != nil { // A
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
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x1E: 0x30}), output) // A -> B

	// An output failure makes the event unhandled while preserving the error.
	handled, err := keyboard.Handle(Event{ScanCode: 0x1E}) // A
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle key = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if want := []Event{{ScanCode: 0x30}}; !reflect.DeepEqual(output.events, want) { // B
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesThroughReleaseWithoutPress(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x1E: 0x30}), output) // A -> B

	// A release without a tracked press must pass through unchanged.
	handled, err := keyboard.Handle(Event{ScanCode: 0x1E, KeyUp: true}) // A
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
