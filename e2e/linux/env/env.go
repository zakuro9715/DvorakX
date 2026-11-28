//go:build linux && e2e

// Package env provides helpers for Linux keyboard end-to-end tests.
package env

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/zakuro9715/DvorakZ/internal/keys"
)

const (
	sourceName = "DvorakZ E2E source"
	outputName = "DvorakZ virtual keyboard"
	markerKey  = 0x2ad // KEY_MACRO30
	eventSize  = 24
	evKey      = 1
)

type keyEvent struct {
	code  uint16
	value int32
}

// BuildRemapper builds the DvorakZ executable in a temporary directory.
func BuildRemapper(t *testing.T) string {
	t.Helper()
	exe := filepath.Join(t.TempDir(), "dvorakz")
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "-o", exe, "./cmd/dvorakz")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", output)
	return exe
}

// NewSource creates a uinput keyboard and opens its evdev receiver.
func NewSource(t *testing.T) (int, string, int) {
	t.Helper()
	injector := newSource(t)
	path, receiver := openReceiver(t, sourceName)
	t.Logf("source: %s", path)
	return injector, path, receiver
}

// OpenOutput opens the remapper output for the test keyboard.
func OpenOutput(t *testing.T) int {
	t.Helper()
	path, receiver := openReceiver(t, outputName)
	t.Logf("output: %s", path)
	return receiver
}

func newSource(t *testing.T) int {
	t.Helper()
	fd, err := unix.Open("/dev/uinput", unix.O_WRONLY|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		fd, err = unix.Open("/dev/input/uinput", unix.O_WRONLY|unix.O_CLOEXEC, 0)
	}
	require.NoError(t, err, "open uinput")
	t.Cleanup(func() { _ = unix.Close(fd) })

	require.NoError(t, unix.IoctlSetInt(fd, 0x40045564, evKey), "UI_SET_EVBIT")
	// udev's keyboard classifier checks for the first 31 ordinary keys.
	for code := 1; code <= 31; code++ {
		require.NoError(t, unix.IoctlSetInt(fd, 0x40045565, code), "UI_SET_KEYBIT")
	}
	for _, code := range []int{keys.C, markerKey} {
		require.NoError(t, unix.IoctlSetInt(fd, 0x40045565, code), "UI_SET_KEYBIT")
	}

	var setup [92]byte
	binary.NativeEndian.PutUint16(setup[:2], 6) // BUS_VIRTUAL
	copy(setup[8:], sourceName)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), 0x405c5503, uintptr(unsafe.Pointer(&setup))) // UI_DEV_SETUP
	require.Zero(t, errno, "UI_DEV_SETUP: %v", errno)
	require.NoError(t, unix.IoctlSetInt(fd, 0x5501, 0), "UI_DEV_CREATE")
	t.Cleanup(func() { _ = unix.IoctlSetInt(fd, 0x5502, 0) }) // UI_DEV_DESTROY
	return fd
}

func openReceiver(t *testing.T, name string) (string, int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var openErr error
	for time.Now().Before(deadline) {
		paths, err := filepath.Glob("/sys/class/input/event*/device/name")
		require.NoError(t, err)
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil || strings.TrimSpace(string(data)) != name {
				continue
			}
			node := filepath.Join("/dev/input", filepath.Base(filepath.Dir(filepath.Dir(path))))
			fd, err := unix.Open(node, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
			if err != nil {
				openErr = &os.PathError{Op: "open", Path: node, Err: err}
				continue
			}
			if !hasMarker(fd) {
				_ = unix.Close(fd)
				continue
			}
			t.Cleanup(func() { _ = unix.Close(fd) })
			return node, fd
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out opening evdev receiver %q: %v", name, openErr)
	return "", -1
}

func hasMarker(fd int) bool {
	var bits [96]byte
	const request = 0x80604521 // EVIOCGBIT(EV_KEY, 96)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), request, uintptr(unsafe.Pointer(&bits)))
	return errno == 0 && bits[markerKey/8]&(1<<(markerKey%8)) != 0
}

// StartRemapper starts DvorakZ and waits for it to grab the source device.
func StartRemapper(t *testing.T, exe, source, configText string) {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(config, []byte(configText), 0o600))
	log, err := os.Create(filepath.Join(dir, "dvorakz.log"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = log.Close() })

	cmd := exec.Command(exe, "--config", config, "--debug")
	cmd.Stdout, cmd.Stderr = log, log
	require.NoError(t, cmd.Start())
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		default:
			_ = cmd.Process.Signal(unix.SIGTERM)
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				_ = cmd.Process.Kill()
				<-done
			}
		}
		data, err := os.ReadFile(log.Name())
		t.Logf("dvorakz exit: %v; log read: %v\n%s", waitErr, err, data)
	})

	deadline := time.Now().Add(10 * time.Second)
	ready := "dvorakz: debug: remapping " + source + "\n"
	for time.Now().Before(deadline) {
		select {
		case <-done:
			t.Fatalf("dvorakz exited before grabbing %s: %v", source, waitErr)
		default:
		}
		data, err := os.ReadFile(log.Name())
		require.NoError(t, err)
		if strings.Contains(string(data), ready) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for dvorakz to grab %s", source)
}

// AssertKeys taps the input keys and checks the output edges in order.
// Pass -1 as silent when there is no second receiver to monitor.
func AssertKeys(t *testing.T, injector, receiver, silent int, input, expected []uint16) {
	t.Helper()
	for _, code := range input {
		sendKey(t, injector, code)
	}
	want := make([]keyEvent, 0, 2*len(expected))
	for _, code := range expected {
		want = append(want, keyEvent{code, 1}, keyEvent{code, 0})
	}
	expectEvents(t, receiver, silent, want)
}

func sendKey(t *testing.T, fd int, code uint16) {
	t.Helper()
	var events [4 * eventSize]byte
	// Each key edge is followed by an EV_SYN/SYN_REPORT frame.
	for i, value := range []uint32{1, 0} {
		event := events[i*2*eventSize:]
		binary.NativeEndian.PutUint16(event[16:18], evKey)
		binary.NativeEndian.PutUint16(event[18:20], code)
		binary.NativeEndian.PutUint32(event[20:24], value)
	}
	n, err := unix.Write(fd, events[:])
	require.NoError(t, err)
	require.Equal(t, len(events), n, "short uinput write")
}

func expectEvents(t *testing.T, receiver, silent int, want []keyEvent) {
	t.Helper()
	fds := []unix.PollFd{{Fd: int32(receiver), Events: unix.POLLIN}, {Fd: int32(silent), Events: unix.POLLIN}}
	got := make([]keyEvent, 0, len(want))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		_, err := unix.Poll(fds, 20)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		require.NoError(t, err)
		require.Zero(t, fds[1].Revents, "grabbed source received input or disconnected")
		require.Zero(t, fds[0].Revents&^unix.POLLIN, "receiver disconnected")
		if fds[0].Revents == 0 {
			continue
		}
		next := readEvents(t, receiver)
		complete := len(got) == len(want)
		got = append(got, next...)
		require.LessOrEqual(t, len(got), len(want), "received extra events: %v; want %v", got, want)
		require.Equal(t, want[:len(got)], got)
		if !complete && len(got) == len(want) {
			deadline = time.Now().Add(100 * time.Millisecond)
		}
	}
	require.Equal(t, want, got)
}

func readEvents(t *testing.T, fd int) []keyEvent {
	t.Helper()
	var data [32 * eventSize]byte
	n, err := unix.Read(fd, data[:])
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return nil
	}
	require.NoError(t, err)
	require.Positive(t, n)
	require.Zero(t, n%eventSize)
	var events []keyEvent
	for offset := 0; offset < n; offset += eventSize {
		event := data[offset : offset+eventSize]
		kind := binary.NativeEndian.Uint16(event[16:18])
		code := binary.NativeEndian.Uint16(event[18:20])
		require.False(t, kind == 0 && code == 3, "evdev reported SYN_DROPPED")
		if kind == evKey {
			events = append(events, keyEvent{code, int32(binary.NativeEndian.Uint32(event[20:24]))})
		}
	}
	return events
}
