//go:build windows && e2e

package env

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	user32          = syscall.NewLazyDLL("user32.dll")
	createWindow    = user32.NewProc("CreateWindowExW")
	destroyWindow   = user32.NewProc("DestroyWindow")
	setForeground   = user32.NewProc("SetForegroundWindow")
	getForeground   = user32.NewProc("GetForegroundWindow")
	setFocus        = user32.NewProc("SetFocus")
	peekMessage     = user32.NewProc("PeekMessageW")
	dispatchMessage = user32.NewProc("DispatchMessageW")
	sendInput       = user32.NewProc("SendInput")
)

type keyboardMessage struct {
	window         uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	x, y           int32
	private        uint32
}

type keyboardInput struct {
	kind                 uint32
	_                    uint32
	virtualKey, scanCode uint16
	flags, time          uint32
	extraInfo            uintptr
	_                    [8]byte
}

var (
	_ [unsafe.Sizeof(keyboardInput{}) - 40]byte
	_ [40 - unsafe.Sizeof(keyboardInput{})]byte
)

type keyEvent struct {
	scanCode uint16
	up       bool
}

func BuildRemapper(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	exe := filepath.Join(tempDir, "dvorakz.exe")
	buildCtx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(buildCtx, "go", "build", "-o", exe, "./cmd/dvorakz")
	build.Dir = filepath.Join("..", "..")
	t.Logf("building dvorakz from %s", build.Dir)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build dvorakz: %v\n%s", err, output)
	}
	t.Logf("built dvorakz: %s", exe)
	return exe
}

func NewReceiver(t *testing.T) uintptr {
	t.Helper()

	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	class, err := syscall.UTF16PtrFromString("EDIT")
	if err != nil {
		t.Fatal(err)
	}
	title, err := syscall.UTF16PtrFromString("DvorakZ E2E")
	if err != nil {
		t.Fatal(err)
	}
	const visibleOverlappedWindow = 0x10CF0000
	window, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)),
		visibleOverlappedWindow, 0, 0, 320, 120, 0, 0, 0, 0)
	if window == 0 {
		t.Fatalf("create receiver window: %v", err)
	}
	t.Cleanup(func() { _, _, _ = destroyWindow.Call(window) })
	_, _, _ = setForeground.Call(window)
	_, _, _ = setFocus.Call(window)

	foreground, _, _ := getForeground.Call()
	t.Logf("created receiver window %#x (foreground %#x)", window, foreground)
	return window
}

func StartRemapper(t *testing.T, exe, configText string) {
	t.Helper()
	config := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(config, []byte(configText), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(exe, "--config", config, "--debug")
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("starting dvorakz with config %s", config)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{})
	logDone := make(chan struct{})
	t.Cleanup(func() {
		t.Logf("stopping dvorakz (PID %d)", cmd.Process.Pid)
		_ = cmd.Process.Kill()
		<-logDone
		_ = cmd.Wait()
	})
	go func() {
		defer close(logDone)
		scanner := bufio.NewScanner(stderr)
		signaled := false
		for scanner.Scan() {
			line := scanner.Text()
			t.Logf("dvorakz: %s", line)
			if !signaled && strings.Contains(line, "keyboard hook installed") {
				signaled = true
				close(ready)
			}
		}
		if err := scanner.Err(); err != nil {
			t.Logf("read dvorakz stderr: %v", err)
		}
	}()
	t.Logf("waiting for keyboard hook (PID %d)", cmd.Process.Pid)
	select {
	case <-ready:
		t.Log("keyboard hook ready")
	case <-logDone:
		t.Fatal("dvorakz exited before the keyboard hook was ready")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the keyboard hook")
	}
}

func AssertKey(t *testing.T, window uintptr, source, destination uint16) {
	t.Helper()

	const (
		scanCodeFlag  = 0x0008
		keyUpFlag     = 0x0002
		inputKeyboard = 1
		wmKeyDown     = 0x0100
		wmKeyUp       = 0x0101
		pmRemove      = 0x0001
	)
	if foreground, _, _ := getForeground.Call(); foreground != window {
		t.Fatalf("receiver %#x is not in the foreground (foreground %#x)", window, foreground)
	}
	t.Logf("sending press and release for scan code %#02x, expecting %#02x", source, destination)
	inputs := [...]keyboardInput{
		{kind: inputKeyboard, scanCode: source, flags: scanCodeFlag},
		{kind: inputKeyboard, scanCode: source, flags: scanCodeFlag | keyUpFlag},
	}
	size := unsafe.Sizeof(inputs[0])
	sent, _, err := sendInput.Call(uintptr(len(inputs)), uintptr(unsafe.Pointer(&inputs[0])), size)
	if sent != uintptr(len(inputs)) {
		if sent == 1 {
			// Release the source key if Windows accepted only the press.
			_, _, _ = sendInput.Call(1, uintptr(unsafe.Pointer(&inputs[1])), size)
		}
		t.Fatalf("SendInput sent %d of %d events: %v", sent, len(inputs), err)
	}

	want := []keyEvent{{scanCode: destination}, {scanCode: destination, up: true}}
	var got []keyEvent
	deadline := time.Now().Add(2 * time.Second)
	var completeAt time.Time
	for time.Now().Before(deadline) {
		var msg keyboardMessage
		for {
			pending, _, _ := peekMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, pmRemove)
			if pending == 0 {
				break
			}
			if msg.window == window && (msg.message == wmKeyDown || msg.message == wmKeyUp) {
				got = append(got, keyEvent{scanCode: uint16(msg.lParam >> 16 & 0xff), up: msg.message == wmKeyUp})
				if len(got) > len(want) || !slices.Equal(got, want[:len(got)]) {
					t.Fatalf("input scan code %#x: got %+v, want %+v", source, got, want)
				}
			}
			_, _, _ = dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
		if len(got) == len(want) {
			if completeAt.IsZero() {
				completeAt = time.Now()
				// Keep pumping briefly to catch delayed duplicate or original events.
				deadline = completeAt.Add(100 * time.Millisecond)
			}
		}
		time.Sleep(time.Millisecond)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("input scan code %#x: got %+v, want %+v", source, got, want)
	}
	t.Logf("received press and release for scan code %#02x", destination)
}
