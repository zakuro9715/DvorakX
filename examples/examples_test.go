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
	if got := loaded.Layers["alternate"].Definition.Text; got != "#src" {
		t.Errorf("alternate definition = %q, want #src", got)
	}

	output := newRecordingOutput()
	mappings, err := keyboard.NewMappingsFromConfig(loaded)
	if err != nil {
		t.Fatalf("NewMappingsFromConfig() error = %v", err)
	}
	simulated := keyboard.NewSimulatedKeyboard(mappings, output)

	// This covers layer remaps, direct global remaps, and the send-keys action.
	for _, event := range []keyboard.Event{
		{VirtualKey: 0x41},              // A
		{VirtualKey: 0x41, KeyUp: true}, // A
		{VirtualKey: 0x43},              // C
		{VirtualKey: 0x43, KeyUp: true}, // C
		{VirtualKey: 0x44},              // D
		{VirtualKey: 0x44, KeyUp: true}, // D
		{VirtualKey: 0x14},              // Caps Lock
		{VirtualKey: 0x14, KeyUp: true}, // Caps Lock
		{VirtualKey: 0x5B},              // Left Windows
		{VirtualKey: 0x5B, KeyUp: true}, // Left Windows
		{VirtualKey: 0x59},              // Y
		{VirtualKey: 0x59, KeyUp: true}, // Y
		{VirtualKey: 0x5A},              // Z
		{VirtualKey: 0x5A, KeyUp: true}, // Z
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
	copyPaste := []keyboard.Event{
		{VirtualKey: 0x11},              // Ctrl
		{VirtualKey: 0x43},              // C
		{VirtualKey: 0x43, KeyUp: true}, // C
		{VirtualKey: 0x56},              // V
		{VirtualKey: 0x56, KeyUp: true}, // V
		{VirtualKey: 0x11, KeyUp: true}, // Ctrl
	}

	want := []keyboard.Event{
		{VirtualKey: 0x42},              // B
		{VirtualKey: 0x42, KeyUp: true}, // B
		{VirtualKey: 0x43},              // C
		{VirtualKey: 0x43, KeyUp: true}, // C
		{VirtualKey: 0x44},              // D
		{VirtualKey: 0x44, KeyUp: true}, // D
		{VirtualKey: 0x11},              // Ctrl
		{VirtualKey: 0x11, KeyUp: true}, // Ctrl
		{VirtualKey: 0x12},              // Alt
		{VirtualKey: 0x12, KeyUp: true}, // Alt
	}
	want = append(want, copyPaste...)
	want = append(want, copyPaste...)
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
