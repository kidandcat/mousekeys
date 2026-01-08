//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>

extern CGEventRef eventCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *refcon);

static CFMachPortRef createEventTap() {
    CGEventMask mask = (1 << kCGEventKeyDown) | (1 << kCGEventKeyUp) | (1 << kCGEventFlagsChanged);
    CFMachPortRef tap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        mask,
        eventCallback,
        NULL
    );
    return tap;
}

static void runEventTap(CFMachPortRef tap) {
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
    CGEventTapEnable(tap, true);
    CFRunLoopRun();
}
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// macOS key codes
const (
	darwinKeyCapsLock = 57
	darwinKeyW        = 13
	darwinKeyA        = 0
	darwinKeyS        = 1
	darwinKeyD        = 2
	darwinKeyQ        = 12
	darwinKeyE        = 14
	darwinKeyZ        = 6
	darwinKeyX        = 7
	darwinKeyR        = 15
	darwinKeyF        = 3
	darwinKeySpace    = 49
	darwinKeyLCtrl    = 59
	darwinKeyLShift   = 56
)

// Global variables for the callback (required by cgo)
var (
	darwinEventChan chan KeyEvent
	darwinHook      *DarwinKeyboardHook
	lastCapsLock    time.Time
	capsLockMu      sync.Mutex
)

// DarwinKeyboardHook implements KeyboardHook for macOS
type DarwinKeyboardHook struct {
	eventChan chan KeyEvent
	running   bool
	mu        sync.Mutex
}

// NewKeyboardHook creates a new keyboard hook for macOS
func NewKeyboardHook() KeyboardHook {
	return &DarwinKeyboardHook{
		eventChan: make(chan KeyEvent, 100),
	}
}

func (h *DarwinKeyboardHook) Start() (<-chan KeyEvent, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return h.eventChan, nil
	}

	// Set global references for callback
	darwinEventChan = h.eventChan
	darwinHook = h
	h.running = true

	go func() {
		tap := C.createEventTap()
		if tap == C.CFMachPortRef(0) {
			fmt.Println("Failed to create event tap. Make sure Accessibility permissions are granted.")
			return
		}
		C.runEventTap(tap)
	}()

	return h.eventChan, nil
}

func (h *DarwinKeyboardHook) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.running = false
	close(h.eventChan)
	return nil
}

// translateKeycode converts macOS keycode to unified Key
func translateKeycode(keycode int64) Key {
	switch keycode {
	case darwinKeyCapsLock:
		return KeyToggle
	case darwinKeyW:
		return KeyMoveUp
	case darwinKeyS:
		return KeyMoveDown
	case darwinKeyA:
		return KeyMoveLeft
	case darwinKeyD:
		return KeyMoveRight
	case darwinKeyQ:
		return KeyDiagUpLeft
	case darwinKeyE:
		return KeyDiagUpRight
	case darwinKeyZ:
		return KeyDiagDownLeft
	case darwinKeyX:
		return KeyDiagDownRight
	case darwinKeySpace:
		return KeyLeftClick
	case darwinKeyLCtrl:
		return KeyRightClick
	case darwinKeyLShift:
		return KeyMiddleClick
	case darwinKeyR:
		return KeyScrollUp
	case darwinKeyF:
		return KeyScrollDown
	default:
		return KeyUnknown
	}
}

//export eventCallback
func eventCallback(proxy C.CGEventTapProxy, eventType C.CGEventType, event C.CGEventRef, refcon unsafe.Pointer) C.CGEventRef {
	keycode := int64(C.CGEventGetIntegerValueField(event, C.kCGKeyboardEventKeycode))
	flags := uint64(C.CGEventGetFlags(event))

	var evt KeyEvent
	evt.RawCode = keycode
	evt.Flags = flags
	evt.Keycode = translateKeycode(keycode)

	// Handle modifier keys via flags changed event
	if eventType == C.kCGEventFlagsChanged {
		evt.EventType = FlagsChanged

		if keycode == darwinKeyCapsLock {
			capsLockMu.Lock()
			if time.Since(lastCapsLock) > 300*time.Millisecond {
				lastCapsLock = time.Now()
				capsLockMu.Unlock()
				if darwinEventChan != nil {
					darwinEventChan <- evt
				}
			} else {
				capsLockMu.Unlock()
			}
			return event
		}

		// Handle Ctrl and Shift when mousekeys is active
		if mc != nil && mc.IsActive() {
			if keycode == darwinKeyLCtrl {
				if flags&(1<<18) != 0 { // Ctrl pressed
					if darwinEventChan != nil {
						darwinEventChan <- evt
					}
				}
				return C.CGEventRef(0)
			}
			if keycode == darwinKeyLShift {
				if flags&(1<<17) != 0 { // Shift pressed
					evt.EventType = KeyDown
				} else { // Shift released
					evt.EventType = KeyUp
				}
				if darwinEventChan != nil {
					darwinEventChan <- evt
				}
				return C.CGEventRef(0)
			}
		}
		return event
	}

	// Handle key down
	if eventType == C.kCGEventKeyDown {
		evt.EventType = KeyDown
		if mc != nil && mc.IsActive() && evt.Keycode != KeyUnknown {
			if darwinEventChan != nil {
				darwinEventChan <- evt
			}
			return C.CGEventRef(0) // Suppress
		}
	}

	// Handle key up
	if eventType == C.kCGEventKeyUp {
		evt.EventType = KeyUp
		if mc != nil && mc.IsActive() && evt.Keycode != KeyUnknown {
			if darwinEventChan != nil {
				darwinEventChan <- evt
			}
			return C.CGEventRef(0) // Suppress
		}
	}

	return event
}
