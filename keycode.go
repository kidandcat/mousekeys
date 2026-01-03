package main

// Key represents a unified key code across platforms
type Key int

const (
	KeyUnknown Key = iota

	// Toggle key
	KeyToggle // CapsLock

	// Movement keys (WASD)
	KeyMoveUp    // W
	KeyMoveDown  // S
	KeyMoveLeft  // A
	KeyMoveRight // D

	// Diagonal movement
	KeyDiagUpLeft    // Q
	KeyDiagUpRight   // E
	KeyDiagDownLeft  // Z
	KeyDiagDownRight // X

	// Actions
	KeyLeftClick   // Space
	KeyRightClick  // Left Ctrl
	KeyMiddleClick // Left Shift
	KeyScrollUp    // R
	KeyScrollDown  // F
)

// KeyEventType represents the type of keyboard event
type KeyEventType int

const (
	KeyDown KeyEventType = iota
	KeyUp
	FlagsChanged // For modifier keys
)

// KeyEvent represents a keyboard event from the platform hook
type KeyEvent struct {
	Keycode   Key
	EventType KeyEventType
	RawCode   int64  // Platform-specific raw keycode
	Flags     uint64 // Platform-specific modifier flags
}
