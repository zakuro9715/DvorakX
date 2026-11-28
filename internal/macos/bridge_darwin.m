//go:build darwin && cgo

#import <AppKit/AppKit.h>
#import "bridge_darwin.h"
#import "_cgo_export.h"

static const int64_t marker = 0x44564B524D4143;
enum { auxiliaryControlButtons = 8, mediaBase = 256, transitionDown = 0x0a, transitionUp = 0x0b };

static CGEventRef receive(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *context) {
    DvorakTap *tap = context;
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        dvorakDisabled(tap->handle);
        CFRunLoopStop(tap->loop);
        return event;
    }
    if (CGEventGetIntegerValueField(event, kCGEventSourceUserData) == marker) {
        return event;
    }

    if (type == NSEventTypeSystemDefined) {
        @autoreleasepool {
            NSEvent *native = [NSEvent eventWithCGEvent:event];
            if (native && native.type == NSEventTypeSystemDefined &&
                native.subtype == auxiliaryControlButtons) {
                uint64_t flags = CGEventGetFlags(event);
                if (dvorakMediaEvent(tap->handle, proxy, (uint32_t)native.data1, &flags)) {
                    return NULL;
                }
                CGEventSetFlags(event, flags);
            }
        }
        return event;
    }

    int code = (int)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
    int up = type == kCGEventKeyUp;
    int repeat = type == kCGEventKeyDown &&
        CGEventGetIntegerValueField(event, kCGKeyboardEventAutorepeat) != 0;

    uint64_t flags = CGEventGetFlags(event);
    if (dvorakEvent(tap->handle, proxy, code, up, repeat, type == kCGEventFlagsChanged, &flags)) {
        return NULL;
    }
    CGEventSetFlags(event, flags);
    return event;
}

int dvorak_open(DvorakTap *tap) {
    if (!CGPreflightListenEventAccess() || !CGPreflightPostEventAccess()) {
        return 1;
    }
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp) |
        CGEventMaskBit(kCGEventFlagsChanged) | CGEventMaskBit(NSEventTypeSystemDefined);
    tap->tap = CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap,
        kCGEventTapOptionDefault, mask, receive, tap);
    if (!tap->tap) {
        return 2;
    }
    tap->source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap->tap, 0);
    if (!tap->source) {
        CFRelease(tap->tap);
        return 2;
    }
    tap->loop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(tap->loop, tap->source, kCFRunLoopDefaultMode);
    return 0;
}

void dvorak_run(DvorakTap *tap) {
    CGEventTapEnable(tap->tap, true);
    CFRunLoopRun();
}

void dvorak_stop(CFRunLoopRef loop) {
    // The block also works when stop arrives before CFRunLoopRun starts.
    CFRunLoopPerformBlock(loop, kCFRunLoopDefaultMode, ^{ CFRunLoopStop(loop); });
    CFRunLoopWakeUp(loop);
}

void dvorak_close(DvorakTap *tap) {
    CFMachPortInvalidate(tap->tap);
    CFRunLoopRemoveSource(tap->loop, tap->source, kCFRunLoopDefaultMode);
    CFRelease(tap->source);
    CFRelease(tap->tap);
}

int dvorak_emit(CGEventTapProxy proxy, int code, int up, int repeat, uint64_t flags) {
    @autoreleasepool {
        CGEventRef event;
        if (code >= mediaBase) {
            NSInteger data = ((code - mediaBase) << 16) |
                ((up ? transitionUp : transitionDown) << 8) | (repeat != 0);
            NSEvent *native = [NSEvent otherEventWithType:NSEventTypeSystemDefined
                location:NSZeroPoint modifierFlags:flags timestamp:0 windowNumber:0
                context:nil subtype:auxiliaryControlButtons data1:data data2:-1];
            event = native.CGEvent ? CGEventCreateCopy(native.CGEvent) : NULL;
        } else {
            event = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)code, !up);
            if (event) {
                CGEventSetIntegerValueField(event, kCGKeyboardEventAutorepeat, repeat);
                switch (code) {
                    case 54: case 55: // Right and left Command
                    case 56: case 60: // Left and right Shift
                    case 58: case 61: // Left and right Option
                    case 59: case 62: // Left and right Control
                    case 57: // Caps Lock
                        CGEventSetType(event, kCGEventFlagsChanged);
                        break;
                }
            }
        }
        if (!event) {
            return 0;
        }
        CGEventSetFlags(event, flags);
        CGEventSetIntegerValueField(event, kCGEventSourceUserData, marker);
        if (proxy && code < mediaBase) {
            CGEventTapPostEvent(proxy, event);
        } else {
            CGEventPost(code >= mediaBase ? kCGHIDEventTap : kCGSessionEventTap, event);
        }
        CFRelease(event);
        return 1;
    }
}
