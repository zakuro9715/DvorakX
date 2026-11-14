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
}

type point struct {
	X int32
	Y int32
}

// Run intercepts and remaps key events until stop is closed.
func Run(mappings *keyboard.Mappings, stop <-chan struct{}, debugOutput io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThreadID.Call()
	debugf(debugOutput, "keyboard hook starting on thread %d", threadID)
	pressed := make(map[uint16]uint16)
	callback := func(code int, wParam uintptr, event *keyboardHook) uintptr {
		if code < 0 {
			return callNext(code, wParam, event)
		}

		if event.ExtraInfo == injectionMarker {
			return callNext(code, wParam, event)
		}

		source := uint16(event.VirtualKey)
		keyUp := event.Flags&0x80 != 0
		var destination uint16
		var ok bool
		if keyUp {
			destination, ok = pressed[source]
			if !ok {
				return callNext(code, wParam, event)
			}
			delete(pressed, source)
		} else {
			destination, ok = pressed[source]
			if !ok {
				destination, ok = mappings.Load()[source]
				if !ok {
					return callNext(code, wParam, event)
				}
				pressed[source] = destination
			}
		}

		flags := uint32(0)
		if isExtendedVirtualKey(destination) {
			flags |= keyeventfExtendedKey
		}
		if keyUp {
			flags |= keyeventfKeyUp
		}
		eventType := "down"
		if flags&keyeventfKeyUp != 0 {
			eventType = "up"
		}
		debugf(debugOutput, "remap vk=0x%02X -> vk=0x%02X (%s)", event.VirtualKey, destination, eventType)
		eventInput := input{
			Type: inputKeyboard,
			Ki: keyboardInput{
				VirtualKey: destination,
				Flags:      flags,
				ExtraInfo:  injectionMarker,
			},
		}
		sent, _, sendErr := procSendInput.Call(1, uintptr(unsafe.Pointer(&eventInput)), unsafe.Sizeof(eventInput))
		if sent == 0 {
			debugf(debugOutput, "SendInput failed for vk=0x%02X: %v", destination, sendErr)
			return callNext(code, wParam, event)
		}
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

func isExtendedVirtualKey(key uint16) bool {
	switch key {
	case 0xA3, 0xA5, 0x5B, 0x5C: // VK_RCONTROL, VK_RMENU, VK_LWIN, VK_RWIN
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
