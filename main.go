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
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
)

const (
	baseSpeed    = 1.0
	maxSpeed     = 100.0
	accelTime    = 1.2
	tickInterval = 16 * time.Millisecond
	scrollAmount = 15
)

// macOS key codes
const (
	KeyCapsLock = 57
	KeyW        = 13
	KeyA        = 0
	KeyS        = 1
	KeyD        = 2
	KeyQ        = 12
	KeyE        = 14
	KeyZ        = 6
	KeyX        = 7
	KeyR        = 15
	KeyF        = 3
	KeySpace    = 49
	KeyLCtrl    = 59
	KeyLShift   = 56
)

type MouseController struct {
	mu            sync.Mutex
	active        bool
	moveStartTime time.Time
	lastDirX      float64
	lastDirY      float64

	keyW, keyA, keyS, keyD bool
	keyQ, keyE, keyZ, keyX bool

	leftDown bool
}

var (
	mc              *MouseController
	lastCapsLock    time.Time
	capsLockMu      sync.Mutex
)

func NewMouseController() *MouseController {
	return &MouseController{}
}

func (mc *MouseController) Toggle() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.active = !mc.active
	if mc.active {
		mc.moveStartTime = time.Time{}
	} else {
		if mc.leftDown {
			robotgo.Toggle("left", "up")
			mc.leftDown = false
		}
		mc.keyW, mc.keyA, mc.keyS, mc.keyD = false, false, false, false
		mc.keyQ, mc.keyE, mc.keyZ, mc.keyX = false, false, false, false
	}
}

func (mc *MouseController) IsActive() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.active
}

func (mc *MouseController) HandleKeyDown(keycode int64) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return false
	}

	switch keycode {
	case KeyW:
		mc.keyW = true
		return true
	case KeyA:
		mc.keyA = true
		return true
	case KeyS:
		mc.keyS = true
		return true
	case KeyD:
		mc.keyD = true
		return true
	case KeyQ:
		mc.keyQ = true
		return true
	case KeyE:
		mc.keyE = true
		return true
	case KeyZ:
		mc.keyZ = true
		return true
	case KeyX:
		mc.keyX = true
		return true
	case KeySpace:
		if !mc.leftDown {
			robotgo.Toggle("left", "down")
			mc.leftDown = true
		}
		return true
	case KeyR:
		robotgo.Scroll(0, scrollAmount)
		return true
	case KeyF:
		robotgo.Scroll(0, -scrollAmount)
		return true
	}
	return false
}

func (mc *MouseController) HandleKeyUp(keycode int64) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return false
	}

	switch keycode {
	case KeyW:
		mc.keyW = false
		return true
	case KeyA:
		mc.keyA = false
		return true
	case KeyS:
		mc.keyS = false
		return true
	case KeyD:
		mc.keyD = false
		return true
	case KeyQ:
		mc.keyQ = false
		return true
	case KeyE:
		mc.keyE = false
		return true
	case KeyZ:
		mc.keyZ = false
		return true
	case KeyX:
		mc.keyX = false
		return true
	case KeySpace:
		if mc.leftDown {
			robotgo.Toggle("left", "up")
			mc.leftDown = false
		}
		return true
	case KeyR:
		return true
	case KeyF:
		return true
	}
	return false
}

func (mc *MouseController) GetMovement() (dx, dy float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return 0, 0
	}

	if mc.keyW {
		dy -= 1
	}
	if mc.keyS {
		dy += 1
	}
	if mc.keyA {
		dx -= 1
	}
	if mc.keyD {
		dx += 1
	}
	if mc.keyQ {
		dx -= 0.707
		dy -= 0.707
	}
	if mc.keyE {
		dx += 0.707
		dy -= 0.707
	}
	if mc.keyZ {
		dx -= 0.707
		dy += 0.707
	}
	if mc.keyX {
		dx += 0.707
		dy += 0.707
	}

	if dx == 0 && dy == 0 {
		mc.moveStartTime = time.Time{}
		mc.lastDirX, mc.lastDirY = 0, 0
		return 0, 0
	}

	dirX, dirY := 0.0, 0.0
	if dx > 0 {
		dirX = 1
	} else if dx < 0 {
		dirX = -1
	}
	if dy > 0 {
		dirY = 1
	} else if dy < 0 {
		dirY = -1
	}

	if dirX != mc.lastDirX || dirY != mc.lastDirY {
		mc.moveStartTime = time.Now()
		mc.lastDirX, mc.lastDirY = dirX, dirY
	}

	if mc.moveStartTime.IsZero() {
		mc.moveStartTime = time.Now()
	}

	elapsed := time.Since(mc.moveStartTime).Seconds()
	progress := elapsed / accelTime
	if progress > 1 {
		progress = 1
	}
	speed := baseSpeed + (maxSpeed-baseSpeed)*progress

	if dx != 0 && dy != 0 {
		dx *= 0.707
		dy *= 0.707
	}

	return dx * speed, dy * speed
}

func (mc *MouseController) RunLoop() {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	screenW, screenH := robotgo.GetScreenSize()

	for range ticker.C {
		dx, dy := mc.GetMovement()
		if dx == 0 && dy == 0 {
			continue
		}

		x, y := robotgo.Location()
		newX := x + int(dx)
		newY := y + int(dy)

		if newX < 0 {
			newX = 0
		} else if newX >= screenW {
			newX = screenW - 1
		}
		if newY < 0 {
			newY = 0
		} else if newY >= screenH {
			newY = screenH - 1
		}

		robotgo.Move(newX, newY)
	}
}

//export eventCallback
func eventCallback(proxy C.CGEventTapProxy, eventType C.CGEventType, event C.CGEventRef, refcon unsafe.Pointer) C.CGEventRef {
	keycode := int64(C.CGEventGetIntegerValueField(event, C.kCGKeyboardEventKeycode))

	// Handle modifier keys via flags changed event
	if eventType == C.kCGEventFlagsChanged {
		if keycode == KeyCapsLock {
			capsLockMu.Lock()
			if time.Since(lastCapsLock) > 300*time.Millisecond {
				lastCapsLock = time.Now()
				capsLockMu.Unlock()
				mc.Toggle()
			} else {
				capsLockMu.Unlock()
			}
			return event
		}

		// Handle Ctrl and Shift when active
		if mc.IsActive() {
			flags := uint64(C.CGEventGetFlags(event))

			if keycode == KeyLCtrl {
				if flags&(1<<18) != 0 { // Ctrl pressed
					robotgo.Click("right", false)
				}
				return C.CGEventRef(0)
			}
			if keycode == KeyLShift {
				if flags&(1<<17) != 0 { // Shift pressed
					robotgo.Click("center", false)
				}
				return C.CGEventRef(0)
			}
		}
		return event
	}

	// Handle other keys when active
	if eventType == C.kCGEventKeyDown {
		if mc.HandleKeyDown(keycode) {
			return C.CGEventRef(0) // Suppress event
		}
	} else if eventType == C.kCGEventKeyUp {
		if mc.HandleKeyUp(keycode) {
			return C.CGEventRef(0) // Suppress event
		}
	}

	return event
}

func startEventTap() {
	tap := C.createEventTap()
	if tap == C.CFMachPortRef(0) {
		fmt.Println("Failed to create event tap. Make sure Accessibility permissions are granted.")
		return
	}
	C.runEventTap(tap)
}

func getLaunchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.mousekeys.app.plist")
}

func getAppPath() string {
	exe, _ := os.Executable()
	// If running from .app bundle, return the .app path
	// exe will be like /Applications/MouseKeys.app/Contents/MacOS/mousekeys
	if idx := strings.Index(exe, ".app/"); idx != -1 {
		return exe[:idx+4] // Include ".app"
	}
	return exe
}

func isRunOnLoginEnabled() bool {
	_, err := os.Stat(getLaunchAgentPath())
	return err == nil
}

func enableRunOnLogin() error {
	appPath := getAppPath()
	var plist string

	if strings.HasSuffix(appPath, ".app") {
		// Use open command for .app bundles
		plist = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mousekeys.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/open</string>
        <string>-a</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, appPath)
	} else {
		// Direct binary execution
		plist = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mousekeys.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, appPath)
	}

	return os.WriteFile(getLaunchAgentPath(), []byte(plist), 0644)
}

func disableRunOnLogin() error {
	return os.Remove(getLaunchAgentPath())
}

func onReady() {
	systray.SetTitle("⌨️")
	systray.SetTooltip("MouseKeys - Caps Lock to toggle")

	mStatus := systray.AddMenuItem("Inactive", "Current status")
	mStatus.Disable()
	systray.AddSeparator()
	mRunOnLogin := systray.AddMenuItem("Run on Login", "Start MouseKeys when you log in")
	if isRunOnLoginEnabled() {
		mRunOnLogin.Check()
	}
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit MouseKeys")

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			if mc.IsActive() {
				mStatus.SetTitle("● Active")
				systray.SetTitle("🖱️")
			} else {
				mStatus.SetTitle("○ Inactive")
				systray.SetTitle("⌨️")
			}
		}
	}()

	go func() {
		for {
			<-mRunOnLogin.ClickedCh
			if mRunOnLogin.Checked() {
				disableRunOnLogin()
				mRunOnLogin.Uncheck()
			} else {
				enableRunOnLogin()
				mRunOnLogin.Check()
			}
		}
	}()

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {}

func main() {
	fmt.Println("MouseKeys - Caps Lock to toggle")
	fmt.Println("WASD/QEZX=move, Space=click, Ctrl=right, Shift=middle, R/F=scroll")

	mc = NewMouseController()

	go mc.RunLoop()
	go startEventTap()

	systray.Run(onReady, onExit)
}
