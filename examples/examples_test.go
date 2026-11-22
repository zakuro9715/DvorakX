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
		{ScanCode: 0x1E},                // A
		{ScanCode: 0x1E, KeyUp: true},   // A
		{ScanCode: 0x3A},                // Caps Lock
		{ScanCode: 0x3A, KeyUp: true},   // Caps Lock
		{ScanCode: 0xE05B},              // Left Windows
		{ScanCode: 0xE05B, KeyUp: true}, // Left Windows
	} {
		if handled, err := simulated.Handle(event); !handled || err != nil {
			t.Fatalf("Handle(%#v) = (%t, %v), want (true, nil)", event, handled, err)
		}
	}

	// The layer and global actions both define noop, so these events are consumed.
	for _, event := range []keyboard.Event{
		{ScanCode: 0x30},              // B
		{ScanCode: 0x30, KeyUp: true}, // B
		{ScanCode: 0x2D},              // X
		{ScanCode: 0x2D, KeyUp: true}, // X
		{ScanCode: 0x2C},              // Z
		{ScanCode: 0x2C, KeyUp: true}, // Z
	} {
		if handled, err := simulated.Handle(event); !handled || err != nil {
			t.Fatalf("Handle(%#v) = (%t, %v), want (true, nil)", event, handled, err)
		}
	}

	want := []keyboard.Event{
		{ScanCode: 0x30},              // B
		{ScanCode: 0x30, KeyUp: true}, // B
		{ScanCode: 0x1D},              // Ctrl
		{ScanCode: 0x1D, KeyUp: true}, // Ctrl
		{ScanCode: 0x38},              // Alt
		{ScanCode: 0x38, KeyUp: true}, // Alt
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
