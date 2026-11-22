//go:build windows && (amd64 || arm64)

// Package testenv provides the Windows environment used by keyboard-hook E2E tests.
package testenv

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/zakuro9715/DvorakX/internal/keys"
)

const (
	inputKeyboard = 1
	keyEventKeyUp = 0x0002

	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	swShow             = 5
	pmRemove           = 0x0001
)

var (
	e2eMutex                sync.Mutex
	user32                  = syscall.NewLazyDLL("user32.dll")
	procCreateWindowExW     = user32.NewProc("CreateWindowExW")
	procDestroyWindow       = user32.NewProc("DestroyWindow")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procPeekMessageW        = user32.NewProc("PeekMessageW")
	procSendInput           = user32.NewProc("SendInput")
	procSetFocus            = user32.NewProc("SetFocus")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procSetWindowTextW      = user32.NewProc("SetWindowTextW")
	procShowWindow          = user32.NewProc("ShowWindow")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
)

type keyboardInput struct {
	virtualKey uint16
	scanCode   uint16
	flags      uint32
	time       uint32
	extraInfo  uintptr
}

type input struct {
	inputType uint32
	_         uint32
	keyboard  keyboardInput
	_         [8]byte
}

type point struct {
	x int32
	y int32
}

type message struct {
	window  uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	point   point
}

// Environment owns an active remapper process and its focused input window.
type Environment struct {
	t      *testing.T
	window uintptr
}

// Start builds and runs DvorakX with config, then creates a focused input window.
func Start(t *testing.T, config string) *Environment {
	t.Helper()
	if os.Getenv("DVORAKX_E2E") != "1" {
		t.Skip("set DVORAKX_E2E=1 to run the Windows keyboard-hook E2E test")
	}

	e2eMutex.Lock()
	t.Cleanup(e2eMutex.Unlock)
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	environment := &Environment{t: t}
	environment.window = createReceiverWindow(t)
	t.Cleanup(func() {
		procDestroyWindow.Call(environment.window)
	})
	focusWindow(t, environment.window)
	startRemapper(t, config)
	return environment
}

// AssertKey verifies that one source key produces exactly one destination character.
func (environment *Environment) AssertKey(source, want string) {
	environment.assertKey(source, want, false)
}

// AssertShiftedKey verifies that Shift plus a source key produces one destination character.
func (environment *Environment) AssertShiftedKey(source, want string) {
	environment.assertKey(source, want, true)
}

func (environment *Environment) assertKey(source, want string, shifted bool) {
	environment.t.Helper()

	virtualKey := virtualKey(environment.t, source)
	empty, err := syscall.UTF16PtrFromString("")
	if err != nil {
		environment.t.Fatalf("encode empty receiver text: %v", err)
	}
	if result, _, err := procSetWindowTextW.Call(environment.window, uintptr(unsafe.Pointer(empty))); result == 0 {
		environment.t.Fatalf("clear receiver text: %v", err)
	}
	if shifted {
		sendKey(environment.t, keys.VKShift, false)
	}
	sendKey(environment.t, virtualKey, false)
	sendKey(environment.t, virtualKey, true)
	if shifted {
		sendKey(environment.t, keys.VKShift, true)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pumpMessages()
		if got := windowText(environment.t, environment.window); got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	environment.t.Errorf("input %q produced %q, want %q", source, windowText(environment.t, environment.window), want)
}

func createReceiverWindow(t *testing.T) uintptr {
	t.Helper()

	className, err := syscall.UTF16PtrFromString("EDIT")
	if err != nil {
		t.Fatalf("encode receiver class name: %v", err)
	}
	title, err := syscall.UTF16PtrFromString("DvorakX E2E receiver")
	if err != nil {
		t.Fatalf("encode receiver title: %v", err)
	}
	window, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow|wsVisible,
		0, 0, 300, 100,
		0, 0, 0, 0,
	)
	if window == 0 {
		t.Fatalf("create receiver window: %v", err)
	}
	procShowWindow.Call(window, swShow)
	return window
}

func focusWindow(t *testing.T, window uintptr) {
	t.Helper()

	if result, _, err := procSetForegroundWindow.Call(window); result == 0 {
		t.Fatalf("focus receiver window: %v", err)
	}
	procSetFocus.Call(window)
}

func startRemapper(t *testing.T, config string) {
	t.Helper()

	root := repositoryRoot(t)
	tempDirectory := t.TempDir()
	executablePath := filepath.Join(tempDirectory, "dvorakx.exe")
	build := exec.Command("go", "build", "-o", executablePath, "./cmd/dvorakx")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build remapper: %v\n%s", err, output)
	}

	configPath := filepath.Join(tempDirectory, "config.toml")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatalf("write test configuration: %v", err)
	}

	command := exec.Command(executablePath, "--config", configPath, "--debug")
	stderr, err := command.StderrPipe()
	if err != nil {
		t.Fatalf("create remapper stderr pipe: %v", err)
	}
	if err := command.Start(); err != nil {
		t.Fatalf("start remapper: %v", err)
	}
	t.Cleanup(func() {
		stopRemapper(t, command)
	})

	installed := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "keyboard hook installed") {
				close(installed)
				return
			}
		}
	}()

	select {
	case <-installed:
	case <-time.After(5 * time.Second):
		t.Fatal("remapper did not install its keyboard hook within 5 seconds")
	}
}

func stopRemapper(t *testing.T, command *exec.Cmd) {
	t.Helper()

	if command.Process == nil {
		return
	}
	if err := command.Process.Kill(); err != nil && err != os.ErrProcessDone {
		t.Errorf("stop remapper: %v", err)
	}
	if err := command.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			t.Errorf("wait for remapper: %v", err)
		}
	}
}

func sendKey(t *testing.T, virtualKey uint16, keyUp bool) {
	t.Helper()

	flags := uint32(0)
	if keyUp {
		flags = keyEventKeyUp
	}
	event := input{
		inputType: inputKeyboard,
		keyboard:  keyboardInput{virtualKey: virtualKey, flags: flags},
	}
	sent, _, err := procSendInput.Call(1, uintptr(unsafe.Pointer(&event)), unsafe.Sizeof(event))
	if sent != 1 {
		t.Fatalf("send key %#x (up=%t): %v", virtualKey, keyUp, err)
	}
}

func pumpMessages() {
	var message message
	for {
		result, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&message)),
			0, 0, 0, pmRemove,
		)
		if result == 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}

func windowText(t *testing.T, window uintptr) string {
	t.Helper()

	buffer := make([]uint16, 32)
	length, _, err := procGetWindowTextW.Call(
		window,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if length == 0 && err != syscall.Errno(0) {
		t.Fatalf("read receiver text: %v", err)
	}
	return syscall.UTF16ToString(buffer)
}

func virtualKey(t *testing.T, name string) uint16 {
	t.Helper()
	key, ok := keys.Lookup(name)
	if !ok {
		t.Fatalf("unknown source key %q", name)
	}
	return key
}

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate E2E test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
