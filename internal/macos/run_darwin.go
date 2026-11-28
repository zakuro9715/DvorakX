//go:build darwin && cgo

package macos

/*
#cgo CFLAGS: -fblocks
#cgo LDFLAGS: -framework ApplicationServices -framework AppKit
#include <stdlib.h>
#include "bridge_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"runtime/cgo"
	"unsafe"

	"github.com/zakuro9715/DvorakZ/internal/keyboard"
)

const capsCode = 57 // Caps Lock

type session struct {
	*remapper
	proxy        C.CGEventTapProxy
	flags        uint64
	caps         bool
	suppressCaps bool
	failed       bool
	disabled     bool
}

// Run remaps keyboard events in the active macOS session until stop is closed.
func Run(mappings *keyboard.Mappings, stop <-chan struct{}, debug io.Writer) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	s := &session{}
	s.remapper = newRemapper(mappings, func(code int, up, repeat bool) {
		if code == capsCode && !up && !repeat {
			s.caps = !s.caps
		}
		if C.dvorak_emit(s.proxy, C.int(code), cbool(up), cbool(repeat), C.uint64_t(s.outputFlags(s.flags, s.suppressCaps, s.caps))) == 0 {
			s.failed = true
		}
	})
	handle := cgo.NewHandle(s)
	defer handle.Delete()
	tap := (*C.DvorakTap)(C.calloc(1, C.size_t(C.sizeof_DvorakTap)))
	if tap == nil {
		return errors.New("allocate macOS keyboard event tap")
	}
	defer C.free(unsafe.Pointer(tap))
	tap.handle = C.uintptr_t(handle)
	switch C.dvorak_open(tap) {
	case 1:
		return errors.New("enable Input Monitoring and Accessibility for DvorakZ in System Settings > Privacy & Security, then restart it")
	case 2:
		return errors.New("create macOS keyboard event tap in the active login session")
	}
	if debug != nil {
		_, _ = fmt.Fprintln(debug, "dvorakz: debug: macOS keyboard event tap installed")
	}
	done, joined := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(joined)
		select {
		case <-stop:
			C.dvorak_stop(tap.loop)
		case <-done:
		}
	}()
	C.dvorak_run(tap)
	close(done)
	<-joined
	C.dvorak_close(tap)
	s.proxy = nil
	s.release()
	if s.disabled {
		return errors.New("macOS disabled the keyboard event tap")
	}
	if s.failed {
		return errors.New("could not create a macOS keyboard event")
	}
	return nil
}

func cbool(value bool) C.int {
	if value {
		return 1
	}
	return 0
}

//export dvorakDisabled
func dvorakDisabled(handle C.uintptr_t) {
	cgo.Handle(handle).Value().(*session).disabled = true
}

//export dvorakEvent
func dvorakEvent(handle C.uintptr_t, proxy C.CGEventTapProxy, rawCode, up, repeat, changed C.int, flags *C.uint64_t) C.int {
	s := cgo.Handle(handle).Value().(*session)
	s.proxy, s.flags = proxy, uint64(*flags)
	code := int(rawCode)
	keyUp := up != 0
	if changed != 0 {
		if code == capsCode {
			s.suppressCaps = s.resolve(code).mapped
			if s.suppressCaps {
				s.handle(code, false, false)
				s.handle(code, true, false)
				return 1
			}
			*flags = C.uint64_t(s.outputFlags(s.flags, false, s.caps))
			return 0
		}
		side := modifierSide(code)
		if side == 0 {
			*flags = C.uint64_t(s.outputFlags(s.flags, s.suppressCaps, s.caps))
			return 0
		}
		keyUp = s.flags&side == 0
	}
	handled := s.handle(code, keyUp, repeat != 0)
	*flags = C.uint64_t(s.outputFlags(s.flags, s.suppressCaps, s.caps))
	return cbool(handled)
}

//export dvorakMediaEvent
func dvorakMediaEvent(handle C.uintptr_t, proxy C.CGEventTapProxy, data C.uint32_t, flags *C.uint64_t) C.int {
	code, up, repeat, ok := mediaKey(uint32(data))
	if !ok {
		return 0
	}
	return dvorakEvent(handle, proxy, C.int(code), cbool(up), cbool(repeat), 0, flags)
}
