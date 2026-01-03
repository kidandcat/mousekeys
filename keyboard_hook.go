package main

// KeyboardHook is the interface for platform-specific keyboard hooks
type KeyboardHook interface {
	// Start begins capturing keyboard events
	// Returns a channel that receives KeyEvents
	Start() (<-chan KeyEvent, error)

	// Stop terminates the keyboard hook
	Stop() error
}

// Autostart is the interface for platform-specific autostart functionality
type Autostart interface {
	// IsEnabled returns whether autostart is currently enabled
	IsEnabled() bool

	// Enable sets up the application to start on login
	Enable() error

	// Disable removes the autostart configuration
	Disable() error
}
