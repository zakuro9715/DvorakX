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
	remap, noop, sendKeys := loaded.ResolveMappings()
	if want := map[uint16]uint16{
		0x41: 0x42, // A -> B
		0x43: 0x43, // C -> C
		0x44: 0x44, // D -> D
		0x14: 0x11, // Caps Lock -> Ctrl
		0x5B: 0x12, // Left Windows -> Alt
	}; !reflect.DeepEqual(remap, want) {
		t.Errorf("remap = %#v, want %#v", remap, want)
	}
	if want := map[uint16]bool{
		0x42: true, // B
		0x58: true, // X
	}; !reflect.DeepEqual(noop, want) {
		t.Errorf("noop = %#v, want %#v", noop, want)
	}
	if want := map[uint16][]uint16{
		0x59: {0x11, 0x43, 0x56}, // Y -> Ctrl+C+V
		0x5a: {0x11, 0x43, 0x56}, // Y -> Ctrl+C+V
	}; !reflect.DeepEqual(sendKeys, want) {
		t.Errorf("sendKeys = %#v, want %#v", sendKeys, want)
	}
	if got := loaded.Layers["alternate"].Definition.Text; got != "#src" {
		t.Errorf("alternate definition = %q, want #src", got)
	}

	output := newRecordingOutput()
	simulated := keyboard.NewSimulatedKeyboard(
		keyboard.NewMappingsWithActions(remap, noop, sendKeys),
		output,
	)

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
