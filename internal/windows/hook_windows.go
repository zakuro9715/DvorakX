//go:build windows

package windows

import (
	"fmt"
	"io"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
	"github.com/zakuro9715/DvorakZ/internal/keys"
)

const (
	whKeyboardLL = 13

	wmQuit = 0x0012

	inputKeyboard        = 1
	keyeventfScancode    = 0x0008
	keyeventfExtendedKey = 0x0001
	keyeventfKeyUp       = 0x0002
	injectionMarker      = 0x44564B52
)

type procedure interface{ Addr() uintptr }

var (
	user32                            = syscall.NewLazyDLL("user32.dll")
	kernel32                          = syscall.NewLazyDLL("kernel32.dll")
	procCallNextHookEx      procedure = user32.NewProc("CallNextHookEx")
	procGetMessageW         procedure = user32.NewProc("GetMessageW")
	procPeekMessageW        procedure = user32.NewProc("PeekMessageW")
	procPostThreadMessageW  procedure = user32.NewProc("PostThreadMessageW")
	procSendInput           procedure = user32.NewProc("SendInput")
	procSetWindowsHookExW   procedure = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx procedure = user32.NewProc("UnhookWindowsHookEx")
	procGetCurrentThreadID  procedure = kernel32.NewProc("GetCurrentThreadId")
)

type keyboardInput struct {
	VirtualKey uint16
	ScanCode   uint16
	Flags      uint32
	Time       uint32
	ExtraInfo  uintptr
}

type keyboardHook struct {
	VirtualKey uint32
	ScanCode   uint32
	Flags      uint32
	Time       uint32
	ExtraInfo  uintptr
}

type message struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   point
	_       uint32
}

type point struct {
	X int32
	Y int32
}

type output struct{}

func (o output) Send(event keyboard.Event) error {
	flags := uint32(0)
	inputKey := keyboardInput{ExtraInfo: injectionMarker}
	if event.Key.VirtualKeyCode != 0 {
		inputKey.VirtualKey = event.Key.VirtualKeyCode
	} else {
		flags |= keyeventfScancode
		inputKey.ScanCode = event.Key.ScanCode
		if event.Key.Extended {
			flags |= keyeventfExtendedKey
		}
	}
	if event.KeyUp {
		flags |= keyeventfKeyUp
	}
	inputKey.Flags = flags
	eventInput := input{
		Type: inputKeyboard,
		Ki:   inputKey,
	}
	sent, _, err := syscall.SyscallN(procSendInput.Addr(), 1, uintptr(unsafe.Pointer(&eventInput)), unsafe.Sizeof(eventInput))
	if sent == 0 {
		return err
	}
	return nil
}

// Run intercepts and remaps key events until stop is closed.
func Run(mappings *keyboard.Mappings, stop <-chan struct{}, debugOutput io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := syscall.SyscallN(procGetCurrentThreadID.Addr())
	debugf(debugOutput, "keyboard hook starting on thread %d", threadID)
	simulated := keyboard.NewSimulatedKeyboard(mappings, output{})
	callback := func(rawCode uintptr, wParam uintptr, event *keyboardHook) uintptr {
		code := int(int32(rawCode))
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
		handled, err := simulated.Handle(keyboard.Event{
			Key: keys.New(
				uint16(event.VirtualKey),
				uint16(event.ScanCode),
				event.Flags&keyeventfExtendedKey != 0,
			),
			KeyUp: keyUp,
		})
		if err != nil {
			debugf(debugOutput, "SendInput failed for sc=0x%02X: %v", event.ScanCode, err)
		}
		if !handled {
			return callNext(code, wParam, event)
		}
		debugf(debugOutput, "remap sc=0x%02X (%s)", event.ScanCode, eventType)
		return 1
	}

	var msg message
	if stop != nil {
		// Ensure WM_QUIT can be posted even when stop is already closed.
		_, _, _ = syscall.SyscallN(procPeekMessageW.Addr(), uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	}
	hook, _, err := syscall.SyscallN(procSetWindowsHookExW.Addr(), whKeyboardLL, syscall.NewCallback(callback), 0, 0)
	if hook == 0 {
		return fmt.Errorf("install low-level keyboard hook: %w", err)
	}
	debugf(debugOutput, "keyboard hook installed")
	defer func() {
		_, _, _ = syscall.SyscallN(procUnhookWindowsHookEx.Addr(), hook)
		debugf(debugOutput, "keyboard hook stopped")
	}()

	if stop != nil {
		done := make(chan struct{})
		joined := make(chan struct{})
		go func() {
			defer close(joined)
			select {
			case <-stop:
				_, _, _ = syscall.SyscallN(procPostThreadMessageW.Addr(), threadID, wmQuit, 0, 0)
			case <-done:
			}
		}()
		defer func() {
			close(done)
			<-joined
		}()
	}

	for {
		result, _, err := syscall.SyscallN(procGetMessageW.Addr(), uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(result) {
		case -1:
			return fmt.Errorf("read Windows message: %w", err)
		case 0:
			return nil
		}
	}
}

func callNext(code int, wParam uintptr, event *keyboardHook) uintptr {
	result, _, _ := syscall.SyscallN(procCallNextHookEx.Addr(), 0, uintptr(code), wParam, uintptr(unsafe.Pointer(event)))
	return result
}

func debugf(output io.Writer, format string, args ...any) {
	if output != nil {
		_, _ = fmt.Fprintf(output, "dvorakz: debug: "+format+"\n", args...)
	}
}
