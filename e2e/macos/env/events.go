//go:build darwin && cgo && e2e

// Package env provides helpers for macOS keyboard end-to-end tests.
package env

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

typedef struct {
    unsigned code;
    bool up;
} KeyEvent;

typedef struct {
    CFMachPortRef tap;
    CFRunLoopSourceRef source;
    CFRunLoopRef loop;
    KeyEvent events[32];
    unsigned count;
    bool disabled;
} Receiver;

static CGEventRef observe(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *context) {
    Receiver *receiver = context;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        receiver->disabled = true;
    } else if (receiver->count < 32) {
        receiver->events[receiver->count++] = (KeyEvent){
            CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode),
            type == kCGEventKeyUp,
        };
    }
    return event;
}

static int open_receiver(Receiver *receiver) {
    if (!CGPreflightListenEventAccess()) return 1;
    if (!CGPreflightPostEventAccess()) return 2;
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp);
    receiver->tap = CGEventTapCreate(kCGSessionEventTap, kCGTailAppendEventTap,
        kCGEventTapOptionListenOnly, mask, observe, receiver);
    if (!receiver->tap) return 3;
    receiver->source = CFMachPortCreateRunLoopSource(NULL, receiver->tap, 0);
    if (!receiver->source) {
        CFRelease(receiver->tap);
        return 3;
    }
    receiver->loop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(receiver->loop, receiver->source, kCFRunLoopDefaultMode);
    return 0;
}

static void close_receiver(Receiver *receiver) {
    CFMachPortInvalidate(receiver->tap);
    CFRunLoopRemoveSource(receiver->loop, receiver->source, kCFRunLoopDefaultMode);
    CFRelease(receiver->source);
    CFRelease(receiver->tap);
}

static void pump_receiver(void) {
    CFRunLoopRunInMode(kCFRunLoopDefaultMode, 0.01, true);
}

static bool tap_key(CGKeyCode code) {
    CGEventRef down = CGEventCreateKeyboardEvent(NULL, code, true);
    CGEventRef up = CGEventCreateKeyboardEvent(NULL, code, false);
    if (down && up) {
        CGEventPost(kCGSessionEventTap, down);
        CGEventPost(kCGSessionEventTap, up);
    }
    if (down) CFRelease(down);
    if (up) CFRelease(up);
    return down && up;
}
#include <stdlib.h>
*/
import "C"

import (
	"runtime"
	"slices"
	"testing"
	"time"
	"unsafe"
)

type keyEvent struct {
	code uint16
	up   bool
}

// Receiver observes key events downstream of the remapper.
type Receiver struct {
	native *C.Receiver
}

// NewReceiver creates the session tap on the test goroutine's OS thread.
func NewReceiver(t *testing.T) *Receiver {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	native := (*C.Receiver)(C.calloc(1, C.size_t(C.sizeof_Receiver)))
	if native == nil {
		t.Fatal("allocate macOS E2E receiver")
	}
	t.Cleanup(func() { C.free(unsafe.Pointer(native)) })
	switch C.open_receiver(native) {
	case 1:
		t.Fatal("macOS E2E receiver lacks Input Monitoring permission in the active login session")
	case 2:
		t.Fatal("macOS E2E sender lacks Accessibility permission in the active login session")
	case 3:
		t.Fatal("cannot create macOS E2E receiver tap in the active login session")
	}
	t.Cleanup(func() { C.close_receiver(native) })
	return &Receiver{native: native}
}

// AssertKey sends a key press and release and checks the exact output edges.
func (r *Receiver) AssertKey(t *testing.T, source, destination uint16) {
	t.Helper()
	r.native.count = 0
	if !bool(C.tap_key(C.CGKeyCode(source))) {
		t.Fatalf("create macOS key events for keycode %#x", source)
	}

	want := []keyEvent{{code: destination}, {code: destination, up: true}}
	var got []keyEvent
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		C.pump_receiver()
		if bool(r.native.disabled) {
			t.Fatal("macOS disabled the E2E receiver tap")
		}
		previous := len(got)
		for _, event := range r.native.events[previous:int(r.native.count)] {
			got = append(got, keyEvent{code: uint16(event.code), up: bool(event.up)})
		}
		if len(got) > len(want) || !slices.Equal(got, want[:len(got)]) {
			t.Fatalf("keycode %#x: got %+v, want %+v", source, got, want)
		}
		if previous < len(want) && len(got) == len(want) {
			// Continue pumping to detect delayed originals or duplicate edges.
			deadline = time.Now().Add(100 * time.Millisecond)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("keycode %#x: got %+v, want %+v", source, got, want)
	}
}
