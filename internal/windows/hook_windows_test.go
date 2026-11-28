//go:build windows

package windows

import (
	"bytes"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

type callbackAddress uintptr

func (address callbackAddress) Addr() uintptr { return uintptr(address) }

// Stub the address used by the real syscall. These tests run serially.
func stubProcedure(t *testing.T, slot *procedure, callback any) {
	t.Helper()
	original := *slot
	*slot = callbackAddress(syscall.NewCallback(callback))
	t.Cleanup(func() { *slot = original })
}

func TestSend(t *testing.T) {
	const vkA = 0x41 // A
	// Verify the native INPUT layout and both outcomes of SendInput.
	for _, tc := range []struct {
		name  string
		event keyboard.Event
		want  keyboardInput
		sent  uintptr
	}{
		{"scan press", keyboard.Event{Key: keys.Key{ScanCode: keys.A}}, keyboardInput{ScanCode: keys.A, Flags: keyeventfScancode}, 1},
		{
			"extended release",
			keyboard.Event{Key: keys.Key{ScanCode: keys.ArrowRight, Extended: true}, KeyUp: true},
			keyboardInput{ScanCode: keys.ArrowRight, Flags: keyeventfScancode | keyeventfExtendedKey | keyeventfKeyUp},
			1,
		},
		{
			"virtual key failure",
			keyboard.Event{Key: keys.Key{VirtualKeyCode: vkA, ScanCode: keys.B, Extended: true}, KeyUp: true},
			keyboardInput{VirtualKey: vkA, Flags: keyeventfKeyUp},
			0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			stubProcedure(t, &procSendInput, func(count uintptr, actual *input, size uintptr) uintptr {
				calls++
				want := tc.want
				want.ExtraInfo = injectionMarker
				wantSize := 28 + 3*(unsafe.Sizeof(uintptr(0))-4)
				if count != 1 || size != wantSize || actual.Type != inputKeyboard || actual.Ki != want {
					t.Errorf("unexpected SendInput arguments: count=%d size=%d input=%+v", count, size, actual)
				}
				return tc.sent
			})
			if err := (output{}).Send(tc.event); (err != nil) != (tc.sent == 0) || calls != 1 {
				t.Fatalf("Send returned %v after %d calls", err, calls)
			}
		})
	}
}

const (
	threadID      = 21
	hookID        = 42
	forwardResult = 17
	wmKeyDown     = 0x100
	wmKeyUp       = 0x101
)

func TestRun(t *testing.T) {
	for _, scenario := range []string{"install failure", "read failure", "quit without stop channel", "preclosed stop with events", "escape hold"} {
		t.Run(scenario, func(t *testing.T) { testRunScenario(t, scenario) })
	}
}

func testRunScenario(t *testing.T, scenario string) {
	t.Helper()
	var log bytes.Buffer
	var debug io.Writer = &log
	var stop chan struct{}
	switch scenario {
	case "quit without stop channel":
		debug = nil
	case "preclosed stop with events":
		stop = make(chan struct{})
		close(stop)
	default:
		stop = make(chan struct{})
	}

	var callback uintptr
	peeks, installs, reads, uninstalls := 0, 0, 0, 0
	quit := make(chan uintptr, 1)
	stubProcedure(t, &procGetCurrentThreadID, func() uintptr { return threadID })
	stubProcedure(t, &procPeekMessageW, func(_ *message, _, _, _, _ uintptr) uintptr {
		peeks++
		return 0
	})
	stubProcedure(t, &procSetWindowsHookExW, func(kind, address, module, thread uintptr) uintptr {
		installs++
		if kind != whKeyboardLL || address == 0 || module != 0 || thread != 0 || (stop != nil && peeks != 1) {
			t.Error("unexpected hook registration arguments or missing message queue")
		}
		callback = address
		if scenario == "install failure" {
			return 0
		}
		return hookID
	})
	stubProcedure(t, &procUnhookWindowsHookEx, func(handle uintptr) uintptr {
		uninstalls++
		if handle != hookID {
			t.Errorf("uninstalled handle %d, want %d", handle, hookID)
		}
		return 1
	})
	stubProcedure(t, &procPostThreadMessageW, func(thread, msg, wParam, lParam uintptr) uintptr {
		if thread != threadID || msg != wmQuit || wParam != 0 || lParam != 0 {
			t.Error("unexpected quit message")
		}
		quit <- msg
		return 1
	})

	arrow := keys.Key{ScanCode: keys.ArrowRight, Extended: true}
	mappings := keyboard.NewMappings(map[keys.Key]keys.Key{
		{ScanCode: keys.Escape}: {ScanCode: keys.A},
		arrow:                   {ScanCode: keys.B}, {ScanCode: keys.C}: {ScanCode: keys.D},
	})
	events := stubHookEvents(t)
	if scenario == "escape hold" {
		stubProcedure(t, &procSendInput, func(_ uintptr, input *input, _ uintptr) uintptr {
			events.sends++
			wantFlags := uint32(keyeventfScancode)
			if events.sends == 2 {
				wantFlags |= keyeventfKeyUp
				if uninstalls != 1 {
					t.Error("released the mapped key before removing the hook")
				}
			}
			if input.Ki.ScanCode != keys.A || input.Ki.Flags != wantFlags {
				t.Errorf("unexpected Esc replacement: %+v", input.Ki)
			}
			return 1
		})
	}

	stubProcedure(t, &procGetMessageW, func(_ *message, _, _, _ uintptr) uintptr {
		reads++
		if scenario == "read failure" {
			return 0xffffffff
		}
		if scenario == "preclosed stop with events" && reads == 1 {
			driveHookEvents(t, callback, events)
			return 1
		}
		if scenario == "escape hold" {
			// Hold physical Esc without a key-up and wait for the emergency quit message.
			event := keyboardHook{ScanCode: keys.Escape}
			got, _, _ := syscall.SyscallN(callback, 0, wmKeyDown, uintptr(unsafe.Pointer(&event)))
			if got != 1 {
				t.Error("the physical Esc key press did not remap")
			}
		}
		if scenario == "preclosed stop with events" || scenario == "escape hold" {
			select {
			case <-quit:
			case <-time.After(5 * time.Second):
				t.Error("timed out waiting for WM_QUIT")
			}
		}
		return 0
	})

	newObserver := func() *keyboard.InputObserver {
		return keyboard.NewInputObserver(func() { close(stop) })
	}
	err := Run(mappings, stop, newObserver, debug)
	wantReads, wantUninstalls, wantError := 1, 1, ""
	switch scenario {
	case "install failure":
		wantReads, wantUninstalls, wantError = 0, 0, "install low-level keyboard hook"
	case "read failure":
		wantError = "read Windows message"
	case "escape hold":
		if events.sends != 2 {
			t.Errorf("got %d keyboard outputs, want mapped press and cleanup release", events.sends)
		}
	case "preclosed stop with events":
		wantReads = 2
		checkHookEvents(t, events, log.String())
	}
	if (err == nil) != (wantError == "") || (err != nil && !strings.Contains(err.Error(), wantError)) {
		t.Errorf("Run returned %v, want error %q", err, wantError)
	}
	if installs != 1 || reads != wantReads || uninstalls != wantUninstalls {
		t.Errorf("got %d installs, %d reads and %d uninstalls, want 1, %d and %d", installs, reads, uninstalls, wantReads, wantUninstalls)
	}
}

type hookEvents struct {
	sends, forwards               int
	expectedEvent                 *keyboardHook
	expectedCode, expectedMessage uintptr
}

func stubHookEvents(t *testing.T) *hookEvents {
	t.Helper()
	// Drive the callback installed by Run and observe its native send and forward calls.
	events := new(hookEvents)
	stubProcedure(t, &procCallNextHookEx, func(handle, code, wParam uintptr, event *keyboardHook) uintptr {
		events.forwards++
		if handle != 0 || int32(code) != int32(events.expectedCode) || wParam != events.expectedMessage || event != events.expectedEvent {
			t.Error("next hook received different callback arguments")
		}
		return forwardResult
	})
	stubProcedure(t, &procSendInput, func(_ uintptr, input *input, _ uintptr) uintptr {
		events.sends++
		scan, flags := uint16(keys.B), uint32(keyeventfScancode)
		switch events.sends {
		case 2:
			flags |= keyeventfKeyUp
		case 3:
			scan = keys.D
		}
		if input.Ki.ScanCode != scan || input.Ki.Flags != flags || input.Ki.ExtraInfo != injectionMarker {
			t.Errorf("unexpected remapped input: %+v", input.Ki)
		}
		if events.sends == 3 {
			return 0
		}
		return 1
	})

	return events
}

func driveHookEvents(t *testing.T, callback uintptr, events *hookEvents) {
	t.Helper()
	for _, event := range []struct {
		code, msg uintptr
		key       keyboardHook
		want      uintptr
	}{
		{0xffffffff, wmKeyDown, keyboardHook{}, forwardResult},
		{0, wmKeyDown, keyboardHook{ScanCode: keys.C, ExtraInfo: injectionMarker}, forwardResult},
		{0, wmKeyDown, keyboardHook{ScanCode: keys.A}, forwardResult},
		{0, wmKeyDown, keyboardHook{ScanCode: keys.ArrowRight, Flags: 0x01}, 1},
		{0, wmKeyUp, keyboardHook{ScanCode: keys.ArrowRight, Flags: 0x81}, 1},
		{0, wmKeyDown, keyboardHook{ScanCode: keys.C}, forwardResult},
	} {
		events.expectedCode, events.expectedMessage, events.expectedEvent = event.code, event.msg, &event.key
		got, _, _ := syscall.SyscallN(callback, event.code, event.msg, uintptr(unsafe.Pointer(events.expectedEvent)))
		if got != event.want {
			t.Errorf("callback returned %d, want %d for %+v", got, event.want, event.key)
		}
	}
}

func checkHookEvents(t *testing.T, events *hookEvents, log string) {
	t.Helper()
	if events.sends != 3 || events.forwards != 4 {
		t.Errorf("got %d sends and %d forwards, want 3 and 4", events.sends, events.forwards)
	}
	for _, line := range []string{"keyboard hook installed", "remap sc=0x4D (down)", "remap sc=0x4D (up)", "SendInput failed", "keyboard hook stopped"} {
		if !strings.Contains(log, line) {
			t.Errorf("missing %q in log:\n%s", line, log)
		}
	}
}
