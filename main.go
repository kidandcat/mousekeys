package main

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

const (
	baseSpeed    = 3.0
	maxSpeed     = 50.0
	accelTime    = 1.0
	tickInterval = 16 * time.Millisecond
)

// macOS key codes (rawcode)
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

	keyW, keyA, keyS, keyD bool
	keyQ, keyE, keyZ, keyX bool

	leftDown bool
}

func NewMouseController() *MouseController {
	return &MouseController{}
}

func (mc *MouseController) Toggle() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.active = !mc.active
	if mc.active {
		mc.moveStartTime = time.Time{}
		// Steal focus using AppleScript
		go func() {
			exec.Command("osascript", "-e", `tell application "Finder" to activate`).Run()
			time.Sleep(50 * time.Millisecond)
			robotgo.KeyTap("escape")
		}()
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

func (mc *MouseController) HandleKeyDown(keycode uint16) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return
	}

	switch keycode {
	case KeyW:
		mc.keyW = true
	case KeyA:
		mc.keyA = true
	case KeyS:
		mc.keyS = true
	case KeyD:
		mc.keyD = true
	case KeyQ:
		mc.keyQ = true
	case KeyE:
		mc.keyE = true
	case KeyZ:
		mc.keyZ = true
	case KeyX:
		mc.keyX = true
	case KeySpace:
		if !mc.leftDown {
			robotgo.Toggle("left", "down")
			mc.leftDown = true
		}
	case KeyLCtrl:
		robotgo.Click("right", false)
	case KeyLShift:
		robotgo.Click("center", false)
	case KeyR:
		robotgo.Scroll(0, 5)
	case KeyF:
		robotgo.Scroll(0, -5)
	}
}

func (mc *MouseController) HandleKeyUp(keycode uint16) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return
	}

	switch keycode {
	case KeyW:
		mc.keyW = false
	case KeyA:
		mc.keyA = false
	case KeyS:
		mc.keyS = false
	case KeyD:
		mc.keyD = false
	case KeyQ:
		mc.keyQ = false
	case KeyE:
		mc.keyE = false
	case KeyZ:
		mc.keyZ = false
	case KeyX:
		mc.keyX = false
	case KeySpace:
		if mc.leftDown {
			robotgo.Toggle("left", "up")
			mc.leftDown = false
		}
	}
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
		return 0, 0
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

	var accumX, accumY float64

	for range ticker.C {
		dx, dy := mc.GetMovement()
		if dx == 0 && dy == 0 {
			accumX, accumY = 0, 0
			continue
		}

		accumX += dx
		accumY += dy

		moveX := int(accumX)
		moveY := int(accumY)

		if moveX != 0 || moveY != 0 {
			x, y := robotgo.Location()
			robotgo.Move(x+moveX, y+moveY)
			accumX -= float64(moveX)
			accumY -= float64(moveY)
		}
	}
}

func main() {
	fmt.Println("MouseKeys - Caps Lock to toggle")
	fmt.Println("WASD/QEZX=move, Space=click, Ctrl=right, Shift=middle, R/F=scroll")

	mc := NewMouseController()

	go mc.RunLoop()

	// Debounce Caps Lock - only toggle once per 300ms
	var lastCapsLock time.Time
	var capsLockMu sync.Mutex

	hook.Register(hook.KeyDown, []string{}, func(e hook.Event) {
		if e.Rawcode == KeyCapsLock {
			capsLockMu.Lock()
			if time.Since(lastCapsLock) > 300*time.Millisecond {
				lastCapsLock = time.Now()
				capsLockMu.Unlock()
				mc.Toggle()
			} else {
				capsLockMu.Unlock()
			}
			return
		}
		mc.HandleKeyDown(e.Rawcode)
	})

	hook.Register(hook.KeyUp, []string{}, func(e hook.Event) {
		if e.Rawcode == KeyCapsLock {
			// Also check on KeyUp in case KeyDown was missed
			capsLockMu.Lock()
			if time.Since(lastCapsLock) > 300*time.Millisecond {
				lastCapsLock = time.Now()
				capsLockMu.Unlock()
				mc.Toggle()
			} else {
				capsLockMu.Unlock()
			}
			return
		}
		mc.HandleKeyUp(e.Rawcode)
	})

	s := hook.Start()
	<-hook.Process(s)
}
