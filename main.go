package main

import (
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

const (
	baseSpeed    = 1.5
	maxSpeed     = 50.0
	accelTime    = 1.2
	tickInterval = 16 * time.Millisecond
	scrollAmount = 15
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
	lastDirX      float64
	lastDirY      float64

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
		robotgo.Scroll(0, scrollAmount)
	case KeyF:
		robotgo.Scroll(0, -scrollAmount)
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

var mc *MouseController

func onReady() {
	systray.SetTitle("⌨️")
	systray.SetTooltip("MouseKeys - Caps Lock to toggle")

	mStatus := systray.AddMenuItem("Inactive", "Current status")
	mStatus.Disable()
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit MouseKeys")

	// Update status when toggled
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
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	// Cleanup
}

func startHooks() {
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

func main() {
	fmt.Println("MouseKeys - Caps Lock to toggle")
	fmt.Println("WASD/QEZX=move, Space=click, Ctrl=right, Shift=middle, R/F=scroll")

	mc = NewMouseController()

	go mc.RunLoop()
	go startHooks()

	systray.Run(onReady, onExit)
}
