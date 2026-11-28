//go:build amd64 || arm64

package linux

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/unix"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

type device struct {
	input, output    int
	name             string
	grabbed, dropped bool
	remap            *remapper
	observer         *keyboard.InputObserver
}

// Run manages Linux keyboards until stop is closed.
func Run(mappings *keyboard.Mappings, stop <-chan struct{}, newObserver func() *keyboard.InputObserver, debug io.Writer) error {
	probe, err := openUinput()
	if err != nil {
		return fmt.Errorf("open uinput (check device and write permission): %w", err)
	}
	_ = unix.Close(probe)
	devices := make(map[string]*device)
	failures := make(map[string]string)
	defer func() {
		for _, d := range devices {
			d.close()
		}
	}()
	var nextScan time.Time
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		now := time.Now()
		if !now.Before(nextScan) {
			reconcile(devices, failures, mappings, debug, newObserver)
			nextScan = now.Add(time.Second)
		}
		if err := pollDevices(devices, failures); err != nil {
			return err
		}
	}
}

func reconcile(devices map[string]*device, failures map[string]string, mappings *keyboard.Mappings, debug io.Writer, newObserver func() *keyboard.InputObserver) {
	paths, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		reportFailure(failures, "/dev/input", err)
		return
	}
	present := make(map[string]bool, len(paths))
	for _, path := range paths {
		present[path] = true
		d := devices[path]
		if d == nil {
			d, err = openKeyboard(path, mappings)
			if err != nil {
				reportFailure(failures, path, err)
				continue
			}
			if d == nil {
				delete(failures, path)
				continue
			}
			d.observer = newObserver()
			devices[path] = d
		}
		if d.grabbed || !virtualReady(d.name) {
			continue
		}
		if err := d.grab(); err != nil {
			reportFailure(failures, path, err)
			continue
		}
		if d.grabbed {
			delete(failures, path)
			if debug != nil {
				_, _ = fmt.Fprintf(debug, "dvorakz: debug: remapping %s\n", path)
			}
		}
	}
	for path, d := range devices {
		if !present[path] {
			d.close()
			delete(devices, path)
		}
	}
	for path := range failures {
		if !present[path] {
			delete(failures, path)
		}
	}
}

func openKeyboard(path string, mappings *keyboard.Mappings) (*device, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	supported, keyboardDevice, err := keyboardCapabilities(fd, path)
	if err != nil || !keyboardDevice {
		_ = unix.Close(fd)
		return nil, err
	}
	output, name, err := createVirtual(supported)
	if err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("create virtual keyboard: %w", err)
	}
	return &device{input: fd, output: output, name: name, remap: newRemapper(mappings, func(code uint16, value int32) error {
		return writeKey(output, code, value)
	})}, nil
}

func (d *device) grab() error {
	// Discard events already delivered by the physical keyboard before takeover.
	var buffer [64 * eventSize]byte
	drained := false
	for range 16 {
		n, err := unix.Read(d.input, buffer[:])
		if errors.Is(err, unix.EAGAIN) {
			drained = true
			break
		}
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.EOF
		}
	}
	if !drained {
		return nil
	}
	state, err := pressedKeys(d.input)
	if err != nil || state != (keyBits{}) {
		return err
	}
	if err := unix.IoctlSetInt(d.input, grabRequest, 1); err != nil {
		return err
	}
	state, err = pressedKeys(d.input)
	if err != nil || state != (keyBits{}) {
		_ = unix.IoctlSetInt(d.input, grabRequest, 0)
		return err
	}
	d.grabbed = true
	return nil
}

func pollDevices(devices map[string]*device, failures map[string]string) error {
	fds := make([]unix.PollFd, 0, len(devices))
	paths := make([]string, 0, len(devices))
	for path, d := range devices {
		events := int16(0)
		if d.grabbed {
			events = unix.POLLIN
		}
		fds = append(fds, unix.PollFd{Fd: int32(d.input), Events: events})
		paths = append(paths, path)
	}
	if _, err := unix.Poll(fds, 100); err != nil {
		if errors.Is(err, unix.EINTR) {
			return nil
		}
		return fmt.Errorf("poll keyboards: %w", err)
	}
	for i, fd := range fds {
		path := paths[i]
		d := devices[path]
		switch {
		case fd.Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0:
			d.close()
			delete(devices, path)
		case fd.Revents&unix.POLLIN != 0:
			if err := d.read(); err != nil {
				reportFailure(failures, path, err)
				d.close()
				delete(devices, path)
			}
		}
	}
	return nil
}

func (d *device) read() error {
	var buffer [64 * eventSize]byte
	n, err := unix.Read(d.input, buffer[:])
	if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EINTR) {
		return nil
	}
	if err != nil {
		return err
	}
	if n == 0 || n%eventSize != 0 {
		return fmt.Errorf("invalid evdev read length %d", n)
	}
	for offset := 0; offset < n; offset += eventSize {
		event := buffer[offset : offset+eventSize]
		kind := binary.NativeEndian.Uint16(event[16:18])
		code := binary.NativeEndian.Uint16(event[18:20])
		value := int32(binary.NativeEndian.Uint32(event[20:24]))
		recovering := d.dropped
		if err := d.process(kind, code, value); err != nil {
			return err
		}
		if recovering && !d.dropped {
			// The snapshot supersedes key events already read in this batch.
			break
		}
	}
	return nil
}

func (d *device) process(kind, code uint16, value int32) error {
	switch {
	case kind == evSyn && code == synDropped:
		d.observer.Reset()
		d.dropped = true
	case d.dropped && kind == evSyn && code == synReport:
		state, err := pressedKeys(d.input)
		if err != nil {
			return err
		}
		d.dropped = false
		return d.remap.sync(state)
	case !d.dropped && kind == evKey:
		if key, ok := codeToKey[code]; ok && value >= 0 && value <= 2 {
			d.observer.Observe(keyboard.Event{Key: key, KeyUp: value == 0}, value == 2)
		}
		return d.remap.handle(code, value, false)
	}
	return nil
}

func (d *device) close() {
	d.observer.Reset()
	for code := range d.remap.held {
		_ = writeKey(d.output, code, 0)
	}
	_ = unix.Close(d.input)                   // Closing evdev releases its exclusive grab.
	_ = unix.IoctlSetInt(d.output, 0x5502, 0) // UI_DEV_DESTROY
	_ = unix.Close(d.output)
}

func reportFailure(failures map[string]string, path string, err error) {
	if failures[path] != err.Error() {
		_, _ = fmt.Fprintf(os.Stderr, "dvorakz: %s: %v\n", path, err)
		failures[path] = err.Error()
	}
}
