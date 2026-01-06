package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/getlantern/systray"
	"github.com/go-vgo/robotgo"
)

const (
	baseSpeed        = 1.5   // Slow start for precision
	maxSpeed         = 100.0 // Max speed
	accelTime        = 0.25  // Time to reach max speed
	precisionTime    = 0.05  // First 50ms stays slow for precision
	tickInterval     = 16 * time.Millisecond
	scrollAmount     = 50
	decelerationRate = 0.72  // Friction multiplier per tick (0-1, lower = faster stop)
	stopThreshold    = 0.5   // Stop when velocity below this
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

	// Velocity for deceleration
	velocityX float64
	velocityY float64

	// Saved velocity for deceleration (only used when all keys released)
	savedVelocityX  float64
	savedVelocityY  float64
	hadDiagonal     bool // Track if we had diagonal movement
	prevHadInput    bool // Track if previous frame had any input
}

var (
	mc                  *MouseController
	hook                KeyboardHook
	autostart           Autostart
	accelerationEnabled = true
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
		robotgo.Click("center", false)
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
	case KeyRightClick, KeyMiddleClick, KeyScrollUp, KeyScrollDown:
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
		mc.velocityX = 0
		mc.velocityY = 0
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

	// Check if we have diagonal input
	isDiagonal := inputX != 0 && inputY != 0

	// Keys are pressed - accelerate
	if inputX != 0 || inputY != 0 {
		// Save velocity only when we have diagonal movement
		if isDiagonal {
			mc.savedVelocityX = mc.velocityX
			mc.savedVelocityY = mc.velocityY
			mc.hadDiagonal = true
		} else if mc.hadDiagonal {
			// Direction changed from diagonal to single axis - clear saved velocity
			mc.savedVelocityX = 0
			mc.savedVelocityY = 0
			mc.hadDiagonal = false
		}

		dirX, dirY := 0.0, 0.0
		if inputX > 0 {
			dirX = 1
		} else if inputX < 0 {
			dirX = -1
		}
		if inputY > 0 {
			dirY = 1
		} else if inputY < 0 {
			dirY = -1
		}

		// Only start timing when we begin moving
		if mc.moveStartTime.IsZero() {
			mc.moveStartTime = time.Now()
		} else if dirX != mc.lastDirX || dirY != mc.lastDirY {
			// Direction changed - halve the accumulated time (halves the speed)
			elapsed := time.Since(mc.moveStartTime)
			mc.moveStartTime = time.Now().Add(-elapsed / 2)
		}
		mc.lastDirX, mc.lastDirY = dirX, dirY

		elapsed := time.Since(mc.moveStartTime).Seconds()

		var speed float64
		var effectiveMaxSpeed, effectiveAccelTime, effectivePrecisionTime float64

		if !accelerationEnabled {
			// Subtle acceleration: smaller range, faster times
			effectiveMaxSpeed = 50.0
			effectiveAccelTime = 0.1
			effectivePrecisionTime = 0.03
		} else {
			effectiveMaxSpeed = maxSpeed
			effectiveAccelTime = accelTime
			effectivePrecisionTime = precisionTime
		}

		// Custom acceleration curve: slow precision phase, then fast acceleration
		if elapsed < effectivePrecisionTime {
			// Precision phase: stay at base speed
			speed = baseSpeed
		} else {
			// Acceleration phase: exponential curve for faster ramp-up
			accelElapsed := elapsed - effectivePrecisionTime
			progress := accelElapsed / effectiveAccelTime
			if progress > 1 {
				progress = 1
			}
			// Use quadratic curve for faster acceleration feel
			progress = progress * progress
			speed = baseSpeed + (effectiveMaxSpeed-baseSpeed)*progress
		}

		// Normalize diagonal
		if isDiagonal {
			inputX *= 0.707
			inputY *= 0.707
		}

		// Update velocity
		mc.velocityX = inputX * speed
		mc.velocityY = inputY * speed
		mc.prevHadInput = true

		return mc.velocityX, mc.velocityY
	}

	// No keys pressed - decelerate
	// Restore saved diagonal velocity only when transitioning from input to no input
	if mc.prevHadInput && mc.hadDiagonal && (mc.savedVelocityX != 0 || mc.savedVelocityY != 0) {
		mc.velocityX = mc.savedVelocityX
		mc.velocityY = mc.savedVelocityY
	}
	mc.savedVelocityX = 0
	mc.savedVelocityY = 0
	mc.hadDiagonal = false
	mc.prevHadInput = false

	mc.moveStartTime = time.Time{}
	mc.lastDirX, mc.lastDirY = 0, 0

	// Apply friction (faster when acceleration disabled)
	effectiveDecel := decelerationRate
	if !accelerationEnabled {
		effectiveDecel = 0.5 // Much faster deceleration
	}
	mc.velocityX *= effectiveDecel
	mc.velocityY *= effectiveDecel

	// Stop if below threshold
	if abs(mc.velocityX) < stopThreshold && abs(mc.velocityY) < stopThreshold {
		mc.velocityX = 0
		mc.velocityY = 0
		return 0, 0
	}

	return mc.velocityX, mc.velocityY
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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
	mAcceleration := systray.AddMenuItem("Acceleration", "Toggle mouse acceleration")
	mAcceleration.Check()
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

	go func() {
		for {
			<-mAcceleration.ClickedCh
			if mAcceleration.Checked() {
				accelerationEnabled = false
				mAcceleration.Uncheck()
			} else {
				accelerationEnabled = true
				mAcceleration.Check()
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
