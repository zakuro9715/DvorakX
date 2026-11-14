package keyboard

import (
	"errors"
	"reflect"
	"testing"
)

func TestSimulatedKeyboardRemapsPressedKeyAfterMappingsReload(t *testing.T) {
	t.Parallel()

	mappings := NewMappings(map[uint16]uint16{0x41: 0x42})
	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(mappings, output)

	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41}); !handled || err != nil {
		t.Fatalf("handle key down = (%t, %v), want (true, nil)", handled, err)
	}
	mappings.Replace(map[uint16]uint16{0x41: 0x43})
	if handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true}); !handled || err != nil {
		t.Fatalf("handle key up = (%t, %v), want (true, nil)", handled, err)
	}

	want := []Event{{VirtualKey: 0x42}, {VirtualKey: 0x42, KeyUp: true}}
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesUnmappedEventsThrough(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output)

	handled, err := keyboard.Handle(Event{VirtualKey: 0x43})
	if handled || err != nil {
		t.Fatalf("handle unmapped key = (%t, %v), want (false, nil)", handled, err)
	}
	if len(output.events) != 0 {
		t.Errorf("output events = %#v, want none", output.events)
	}
}

func TestSimulatedKeyboardPassesThroughWhenOutputFails(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("send event")
	output := &recordingOutput{err: wantErr}
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output)

	handled, err := keyboard.Handle(Event{VirtualKey: 0x41})
	if handled || !errors.Is(err, wantErr) {
		t.Fatalf("handle key = (%t, %v), want (false, %v)", handled, err, wantErr)
	}
	if want := []Event{{VirtualKey: 0x42}}; !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

func TestSimulatedKeyboardPassesThroughReleaseWithoutPress(t *testing.T) {
	t.Parallel()

	output := newRecordingOutput()
	keyboard := NewSimulatedKeyboard(NewMappings(map[uint16]uint16{0x41: 0x42}), output)

	handled, err := keyboard.Handle(Event{VirtualKey: 0x41, KeyUp: true})
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
