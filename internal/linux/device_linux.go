//go:build amd64 || arm64

package linux

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	evSyn       = 0
	evKey       = 1
	evRel       = 2
	evAbs       = 3
	synReport   = 0
	synDropped  = 3
	keyMax      = 0x2ff
	eventSize   = 24
	grabRequest = 0x40044590 // EVIOCGRAB
	virtualName = "DvorakZ virtual keyboard"
)

type keyBits [keyMax/8 + 1]byte

func (bits *keyBits) has(code uint16) bool {
	return code <= keyMax && bits[code/8]&(1<<(code%8)) != 0
}

func readBits(fd int, number uintptr, bits []byte) error {
	request := 0x80004500 | uintptr(len(bits))<<16 | number
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), request, uintptr(unsafe.Pointer(&bits[0])))
	if errno != 0 {
		return errno
	}
	return nil
}

func pressedKeys(fd int) (keyBits, error) {
	var bits keyBits
	err := readBits(fd, 0x18, bits[:]) // EVIOCGKEY
	return bits, err
}

func devicePath(fd int) (string, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return "", err
	}
	return fmt.Sprintf("c%d:%d", unix.Major(stat.Rdev), unix.Minor(stat.Rdev)), nil
}

func keyboardCapabilities(fd int, path string) (keyBits, bool, error) {
	var supported keyBits
	id, err := devicePath(fd)
	if err != nil {
		return supported, false, err
	}
	data, err := os.ReadFile(filepath.Join("/run/udev/data", id))
	if errors.Is(err, os.ErrNotExist) {
		return supported, false, nil
	}
	if err != nil {
		return supported, false, err
	}
	if !slices.Contains(strings.Split(string(data), "\n"), "E:ID_INPUT_KEYBOARD=1") {
		return supported, false, nil
	}
	name, err := os.ReadFile(filepath.Join("/sys/class/input", filepath.Base(path), "device/name"))
	if err != nil || strings.TrimSpace(string(name)) == virtualName {
		return supported, false, err
	}
	var types [4]byte
	if err := readBits(fd, 0x20, types[:]); err != nil {
		return supported, false, err
	}
	if types[0]&(1<<evRel|1<<evAbs) != 0 || types[0]&(1<<evKey) == 0 {
		return supported, false, nil
	}
	if err := readBits(fd, 0x20+evKey, supported[:]); err != nil {
		return supported, false, err
	}
	// Mouse buttons can share a node even when motion has a separate node.
	for code := uint16(0x110); code <= 0x117; code++ { // BTN_LEFT through BTN_TASK
		if supported.has(code) {
			return supported, false, nil
		}
	}
	return supported, true, nil
}

func openUinput() (int, error) {
	for _, path := range []string{"/dev/uinput", "/dev/input/uinput"} {
		fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if !errors.Is(err, unix.ENOENT) || path == "/dev/input/uinput" {
			return fd, err
		}
	}
	return -1, unix.ENOENT
}

func createVirtual(supported keyBits) (int, string, error) {
	fd, err := openUinput()
	if err != nil {
		return -1, "", err
	}
	name, err := setupVirtual(fd, supported)
	if err != nil {
		_ = unix.Close(fd)
		return -1, "", err
	}
	return fd, name, nil
}

func setupVirtual(fd int, supported keyBits) (string, error) {
	if err := unix.IoctlSetInt(fd, 0x40045564, evKey); err != nil { // UI_SET_EVBIT
		return "", err
	}
	for code := uint16(1); code <= keyMax; code++ {
		if _, mapped := codeToKey[code]; !supported.has(code) && !mapped {
			continue
		}
		if err := unix.IoctlSetInt(fd, 0x40045565, int(code)); err != nil { // UI_SET_KEYBIT
			return "", err
		}
	}
	var setup [92]byte
	binary.NativeEndian.PutUint16(setup[:2], 6) // BUS_VIRTUAL
	copy(setup[8:], virtualName)
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), 0x405c5503, uintptr(unsafe.Pointer(&setup))) // UI_DEV_SETUP
	if errno != 0 {
		return "", errno
	}
	if err := unix.IoctlSetInt(fd, 0x5501, 0); err != nil { // UI_DEV_CREATE
		return "", err
	}
	var name [64]byte
	_, _, errno = unix.Syscall(unix.SYS_IOCTL, uintptr(fd), 0x8040552c, uintptr(unsafe.Pointer(&name))) // UI_GET_SYSNAME
	if errno != 0 {
		return "", errno
	}
	return strings.TrimRight(string(name[:]), "\x00"), nil
}

func virtualReady(name string) bool {
	entries, err := filepath.Glob(filepath.Join("/sys/devices/virtual/input", name, "event*"))
	if err != nil {
		return false
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(entry, "dev"))
		if err != nil {
			continue
		}
		major, minor, ok := strings.Cut(strings.TrimSpace(string(data)), ":")
		if !ok {
			continue
		}
		if _, err := strconv.Atoi(major); err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join("/run/udev/data", "c"+major+":"+minor)); err == nil {
			return true
		}
	}
	return false
}

func writeKey(fd int, code uint16, value int32) error {
	var events [2 * eventSize]byte
	binary.NativeEndian.PutUint16(events[16:18], evKey)
	binary.NativeEndian.PutUint16(events[18:20], code)
	binary.NativeEndian.PutUint32(events[20:24], uint32(value))
	for {
		n, err := unix.Write(fd, events[:])
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err == nil && n != len(events) {
			return fmt.Errorf("short uinput write: %d bytes", n)
		}
		return err
	}
}
