package examples

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/zakuro9715/DvorakX/internal/config"
	"github.com/zakuro9715/DvorakX/internal/keyboard"
)

func TestAllBehavior(t *testing.T) {
	loaded, err := config.LoadFile(filepath.Join("all.toml"))
	if err != nil {
		t.Fatalf("LoadFile(all.toml) error = %v", err)
	}

	output := newRecordingOutput()
	simulated := keyboard.NewSimulatedKeyboard(
		keyboard.NewMappingsWithActions(loaded.Remap, loaded.Noop),
		output,
	)

	// The layer definition remaps "a" to "b", while caps is a direct remap to ctrl.
	for _, event := range []keyboard.Event{
		{VirtualKey: 0x41},              // A
		{VirtualKey: 0x41, KeyUp: true}, // A
		{VirtualKey: 0x14},              // Caps Lock
		{VirtualKey: 0x14, KeyUp: true}, // Caps Lock
	} {
		if handled, err := simulated.Handle(event); !handled || err != nil {
			t.Fatalf("Handle(%#v) = (%t, %v), want (true, nil)", event, handled, err)
		}
	}

	// The layer and global actions both define noop, so these events are consumed.
	for _, event := range []keyboard.Event{
		{VirtualKey: 0x42},              // B
		{VirtualKey: 0x42, KeyUp: true}, // B
		{VirtualKey: 0x58},              // X
		{VirtualKey: 0x58, KeyUp: true}, // X
	} {
		if handled, err := simulated.Handle(event); !handled || err != nil {
			t.Fatalf("Handle(%#v) = (%t, %v), want (true, nil)", event, handled, err)
		}
	}

	want := []keyboard.Event{
		{VirtualKey: 0x42},              // B
		{VirtualKey: 0x42, KeyUp: true}, // B
		{VirtualKey: 0x11},              // Ctrl
		{VirtualKey: 0x11, KeyUp: true}, // Ctrl
	}
	if !reflect.DeepEqual(output.events, want) {
		t.Errorf("output events = %#v, want %#v", output.events, want)
	}
}

type recordingOutput struct {
	events []keyboard.Event
}

func (o *recordingOutput) Send(event keyboard.Event) error {
	o.events = append(o.events, event)
	return nil
}

func newRecordingOutput() *recordingOutput {
	return new(recordingOutput)
}
