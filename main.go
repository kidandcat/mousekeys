package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
)

const (
	slowSpeed     = 2.0   // Precision speed (shift or first 150ms)
	normalSpeed   = 15.0  // Normal speed
	precisionTime = 0.15  // 150ms precision phase
	tickInterval  = 16 * time.Millisecond
	scrollAmount  = 50
)

type MouseController struct {
	mu            sync.Mutex
	active        bool
	moveStartTime time.Time

	keyW, keyA, keyS, keyD bool
	keyQ, keyE, keyZ, keyX bool

	leftDown  bool
	shiftDown bool // Slow movement modifier
}

var (
	mc              *MouseController
	hook            KeyboardHook
	autostart       Autostart
	speedMultiplier = 1.0
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
		mc.shiftDown = false
	}
}

func (mc *MouseController) SetShiftState(pressed bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.shiftDown = pressed
}

func (mc *MouseController) IsActive() bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.active
}

// HandleKeyDownByKey processes a key press using the unified Key type
func (mc *MouseController) HandleKeyDownByKey(key Key) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return false
	}

	switch key {
	case KeyMoveUp:
		mc.keyW = true
		return true
	case KeyMoveLeft:
		mc.keyA = true
		return true
	case KeyMoveDown:
		mc.keyS = true
		return true
	case KeyMoveRight:
		mc.keyD = true
		return true
	case KeyDiagUpLeft:
		mc.keyQ = true
		return true
	case KeyDiagUpRight:
		mc.keyE = true
		return true
	case KeyDiagDownLeft:
		mc.keyZ = true
		return true
	case KeyDiagDownRight:
		mc.keyX = true
		return true
	case KeyLeftClick:
		if !mc.leftDown {
			robotgo.Toggle("left", "down")
			mc.leftDown = true
		}
		return true
	case KeyRightClick:
		robotgo.Click("right", false)
		return true
	case KeyMiddleClick:
		// Shift is now slow movement modifier
		mc.shiftDown = true
		return true
	case KeyScrollUp:
		robotgo.Scroll(0, scrollAmount)
		return true
	case KeyScrollDown:
		robotgo.Scroll(0, -scrollAmount)
		return true
	}
	return false
}

// HandleKeyUpByKey processes a key release using the unified Key type
func (mc *MouseController) HandleKeyUpByKey(key Key) bool {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return false
	}

	switch key {
	case KeyMoveUp:
		mc.keyW = false
		return true
	case KeyMoveLeft:
		mc.keyA = false
		return true
	case KeyMoveDown:
		mc.keyS = false
		return true
	case KeyMoveRight:
		mc.keyD = false
		return true
	case KeyDiagUpLeft:
		mc.keyQ = false
		return true
	case KeyDiagUpRight:
		mc.keyE = false
		return true
	case KeyDiagDownLeft:
		mc.keyZ = false
		return true
	case KeyDiagDownRight:
		mc.keyX = false
		return true
	case KeyLeftClick:
		if mc.leftDown {
			robotgo.Toggle("left", "up")
			mc.leftDown = false
		}
		return true
	case KeyMiddleClick:
		// Shift released - stop slow movement
		mc.shiftDown = false
		return true
	case KeyRightClick, KeyScrollUp, KeyScrollDown:
		return true
	}
	return false
}

// Legacy HandleKeyDown for backward compatibility with tests (uses raw keycodes)
// This will be removed once all platforms are implemented
func (mc *MouseController) HandleKeyDown(keycode int64) bool {
	// Map legacy macOS keycodes to unified keys for backward compatibility
	key := legacyKeycodeToKey(keycode)
	return mc.HandleKeyDownByKey(key)
}

// Legacy HandleKeyUp for backward compatibility with tests
func (mc *MouseController) HandleKeyUp(keycode int64) bool {
	key := legacyKeycodeToKey(keycode)
	return mc.HandleKeyUpByKey(key)
}

// legacyKeycodeToKey maps old macOS keycodes to unified Key type
// This maintains backward compatibility with existing tests
func legacyKeycodeToKey(keycode int64) Key {
	switch keycode {
	case 57: // CapsLock
		return KeyToggle
	case 13: // W
		return KeyMoveUp
	case 1: // S
		return KeyMoveDown
	case 0: // A
		return KeyMoveLeft
	case 2: // D
		return KeyMoveRight
	case 12: // Q
		return KeyDiagUpLeft
	case 14: // E
		return KeyDiagUpRight
	case 6: // Z
		return KeyDiagDownLeft
	case 7: // X
		return KeyDiagDownRight
	case 49: // Space
		return KeyLeftClick
	case 59: // LCtrl
		return KeyRightClick
	case 56: // LShift
		return KeyMiddleClick
	case 15: // R
		return KeyScrollUp
	case 3: // F
		return KeyScrollDown
	default:
		return KeyUnknown
	}
}

func (mc *MouseController) GetMovement() (dx, dy float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.active {
		return 0, 0
	}

	// Get input direction
	inputX, inputY := 0.0, 0.0
	if mc.keyW {
		inputY -= 1
	}
	if mc.keyS {
		inputY += 1
	}
	if mc.keyA {
		inputX -= 1
	}
	if mc.keyD {
		inputX += 1
	}
	if mc.keyQ {
		inputX -= 0.707
		inputY -= 0.707
	}
	if mc.keyE {
		inputX += 0.707
		inputY -= 0.707
	}
	if mc.keyZ {
		inputX -= 0.707
		inputY += 0.707
	}
	if mc.keyX {
		inputX += 0.707
		inputY += 0.707
	}

	// No movement
	if inputX == 0 && inputY == 0 {
		mc.moveStartTime = time.Time{}
		return 0, 0
	}

	// Start timing when we begin moving
	if mc.moveStartTime.IsZero() {
		mc.moveStartTime = time.Now()
	}

	elapsed := time.Since(mc.moveStartTime).Seconds()

	// Speed selection: slow if shift held or in precision phase
	var speed float64
	if mc.shiftDown || elapsed < precisionTime {
		speed = slowSpeed
	} else {
		speed = normalSpeed
	}
	speed *= speedMultiplier

	// Normalize diagonal
	if inputX != 0 && inputY != 0 {
		inputX *= 0.707
		inputY *= 0.707
	}

	return inputX * speed, inputY * speed
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

// processKeyEvent handles incoming keyboard events from the hook
func processKeyEvent(evt KeyEvent) {
	switch evt.EventType {
	case FlagsChanged:
		if evt.Keycode == KeyToggle {
			mc.Toggle()
		} else if evt.Keycode == KeyRightClick || evt.Keycode == KeyMiddleClick {
			// These are handled in the hook for macOS (need flags check)
			mc.HandleKeyDownByKey(evt.Keycode)
		}
	case KeyDown:
		mc.HandleKeyDownByKey(evt.Keycode)
	case KeyUp:
		mc.HandleKeyUpByKey(evt.Keycode)
	}
}

func onReady() {
	systray.SetTitle("⌨️")
	systray.SetTooltip("MouseKeys - Caps Lock to toggle")

	mStatus := systray.AddMenuItem("Inactive", "Current status")
	mStatus.Disable()
	systray.AddSeparator()

	// Speed submenu
	mSpeed := systray.AddMenuItem("Speed: Normal", "Adjust mouse speed")
	mSpeedSlow := mSpeed.AddSubMenuItem("Slow (50%)", "Half speed")
	mSpeedMedium := mSpeed.AddSubMenuItem("Medium (75%)", "Medium speed")
	mSpeedNormal := mSpeed.AddSubMenuItem("Normal (100%)", "Default speed")
	mSpeedNormal.Check()
	mSpeedFast := mSpeed.AddSubMenuItem("Fast (150%)", "Faster speed")

	mRunOnLogin := systray.AddMenuItem("Run on Login", "Start MouseKeys when you log in")
	if autostart.IsEnabled() {
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

	// Speed selection handlers
	uncheckAllSpeeds := func() {
		mSpeedSlow.Uncheck()
		mSpeedMedium.Uncheck()
		mSpeedNormal.Uncheck()
		mSpeedFast.Uncheck()
	}

	go func() {
		for {
			select {
			case <-mSpeedSlow.ClickedCh:
				speedMultiplier = 0.5
				uncheckAllSpeeds()
				mSpeedSlow.Check()
				mSpeed.SetTitle("Speed: Slow")
			case <-mSpeedMedium.ClickedCh:
				speedMultiplier = 0.75
				uncheckAllSpeeds()
				mSpeedMedium.Check()
				mSpeed.SetTitle("Speed: Medium")
			case <-mSpeedNormal.ClickedCh:
				speedMultiplier = 1.0
				uncheckAllSpeeds()
				mSpeedNormal.Check()
				mSpeed.SetTitle("Speed: Normal")
			case <-mSpeedFast.ClickedCh:
				speedMultiplier = 1.5
				uncheckAllSpeeds()
				mSpeedFast.Check()
				mSpeed.SetTitle("Speed: Fast")
			}
		}
	}()

	go func() {
		for {
			<-mRunOnLogin.ClickedCh
			if mRunOnLogin.Checked() {
				autostart.Disable()
				mRunOnLogin.Uncheck()
			} else {
				autostart.Enable()
				mRunOnLogin.Check()
			}
		}
	}()

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
	}()
}

func onExit() {
	if hook != nil {
		hook.Stop()
	}
}

func main() {
	fmt.Println("MouseKeys - Caps Lock to toggle")
	fmt.Println("WASD/QEZX=move, Space=click, Ctrl=right, Shift=middle, R/F=scroll")

	mc = NewMouseController()
	hook = NewKeyboardHook()
	autostart = NewAutostart()

	go mc.RunLoop()

	// Start keyboard hook and process events
	events, err := hook.Start()
	if err != nil {
		fmt.Printf("Failed to start keyboard hook: %v\n", err)
		return
	}

	go func() {
		for evt := range events {
			processKeyEvent(evt)
		}
	}()

	systray.Run(onReady, onExit)
}
