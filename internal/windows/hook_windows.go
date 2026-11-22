//go:build windows

package windows

import (
	"fmt"
	"io"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/zakuro9715/DvorakX/internal/keyboard"
)

const (
	whKeyboardLL = 13

	wmQuit = 0x0012

	inputKeyboard        = 1
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	keyeventfScancode    = 0x0008
	injectionMarker      = 0x44564B52
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procCallNextHookEx      = user32.NewProc("CallNextHookEx")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procPostThreadMessageW  = user32.NewProc("PostThreadMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procGetCurrentThreadID  = kernel32.NewProc("GetCurrentThreadId")
)

type keyboardInput struct {
	ScanCode  uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type keyboardHook struct {
	ScanCode  uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type message struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   point
}

type point struct {
	X int32
	Y int32
}

type output struct{}

func (o output) Send(event keyboard.Event) error {
	flags := uint32(0)
	if isExtendedScanCode(event.ScanCode) {
		flags |= keyeventfExtendedKey
	}
	flags |= keyeventfScancode
	if event.KeyUp {
		flags |= keyeventfKeyUp
	}
	eventInput := input{
		Type: inputKeyboard,
		Ki: keyboardInput{
			ScanCode:  uint16(event.ScanCode & 0xFF),
			Flags:     flags,
			ExtraInfo: injectionMarker,
		},
	}
	sent, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&eventInput)), unsafe.Sizeof(eventInput))
	if sent == 0 {
		return err
	}
	return nil
}

// Run intercepts and remaps key events until stop is closed.
func Run(mappings *keyboard.Mappings, stop <-chan struct{}, debugOutput io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThreadID.Call()
	debugf(debugOutput, "keyboard hook starting on thread %d", threadID)
	simulated := keyboard.NewSimulatedKeyboard(mappings, output{})
	callback := func(code int, wParam uintptr, event *keyboardHook) uintptr {
		if code < 0 {
			return callNext(code, wParam, event)
		}

		if event.ExtraInfo == injectionMarker {
			return callNext(code, wParam, event)
		}

		keyUp := event.Flags&0x80 != 0
		eventType := "down"
		if keyUp {
			eventType = "up"
		}
		scanCode := uint16(event.ScanCode)
		if event.Flags&keyeventfExtendedKey != 0 {
			scanCode |= 0xE000
		}
		handled, err := simulated.Handle(keyboard.Event{ScanCode: scanCode, KeyUp: keyUp})
		if err != nil {
			debugf(debugOutput, "SendInput failed for sc=0x%04X: %v", scanCode, err)
		}
		if !handled {
			return callNext(code, wParam, event)
		}
		debugf(debugOutput, "remap sc=0x%04X (%s)", scanCode, eventType)
		return 1
	}

	hook, _, err := procSetWindowsHookExW.Call(whKeyboardLL, syscall.NewCallback(callback), 0, 0)
	if hook == 0 {
		return fmt.Errorf("install low-level keyboard hook: %w", err)
	}
	debugf(debugOutput, "keyboard hook installed")
	defer func() {
		procUnhookWindowsHookEx.Call(hook)
		debugf(debugOutput, "keyboard hook stopped")
	}()

	if stop != nil {
		go func() {
			<-stop
			procPostThreadMessageW.Call(threadID, wmQuit, 0, 0)
		}()
	}

	var msg message
	for {
		result, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch result {
		case ^uintptr(0):
			return fmt.Errorf("read Windows message: %w", err)
		case 0:
			return nil
		}
	}
}

func callNext(code int, wParam uintptr, event *keyboardHook) uintptr {
	result, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, uintptr(unsafe.Pointer(event)))
	return result
}

func isExtendedScanCode(scanCode uint16) bool {
	switch scanCode {
	case 0xE01D, 0xE038, 0xE05B, 0xE05C: // RCONTROL, RALT, LWIN, RWIN
		return true
	default:
		return false
	}
}

func debugf(output io.Writer, format string, args ...any) {
	if output != nil {
		fmt.Fprintf(output, "dvorakv: debug: "+format+"\n", args...)
	}
}
