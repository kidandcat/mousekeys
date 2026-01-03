//go:build linux

package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Linux evdev key codes
const (
	linuxKeyCapsLock  = 58
	linuxKeyW         = 17
	linuxKeyA         = 30
	linuxKeyS         = 31
	linuxKeyD         = 32
	linuxKeyQ         = 16
	linuxKeyE         = 18
	linuxKeyZ         = 44
	linuxKeyX         = 45
	linuxKeyR         = 19
	linuxKeyF         = 33
	linuxKeySpace     = 57
	linuxKeyLeftCtrl  = 29
	linuxKeyLeftShift = 42
)

// evdev event types
const (
	EV_KEY = 1
)

// evdev key states
const (
	KEY_RELEASED = 0
	KEY_PRESSED  = 1
	KEY_REPEAT   = 2
)

// InputEvent represents a Linux input event
type InputEvent struct {
	Time  [16]byte // struct timeval
	Type  uint16
	Code  uint16
	Value int32
}

// LinuxKeyboardHook implements KeyboardHook for Linux
type LinuxKeyboardHook struct {
	eventChan chan KeyEvent
	running   bool
	device    *os.File
	stopChan  chan struct{}
}

// NewKeyboardHook creates a new keyboard hook for Linux
func NewKeyboardHook() KeyboardHook {
	return &LinuxKeyboardHook{
		eventChan: make(chan KeyEvent, 100),
		stopChan:  make(chan struct{}),
	}
}

func (h *LinuxKeyboardHook) Start() (<-chan KeyEvent, error) {
	// Find keyboard device
	devicePath, err := findKeyboardDevice()
	if err != nil {
		return nil, fmt.Errorf("failed to find keyboard device: %v", err)
	}

	h.device, err = os.Open(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open keyboard device %s: %v (try running as root or add user to 'input' group)", devicePath, err)
	}

	h.running = true

	go func() {
		defer h.device.Close()

		buf := make([]byte, 24) // sizeof(struct input_event)
		for h.running {
			n, err := h.device.Read(buf)
			if err != nil || n != 24 {
				continue
			}

			event := InputEvent{
				Type:  binary.LittleEndian.Uint16(buf[16:18]),
				Code:  binary.LittleEndian.Uint16(buf[18:20]),
				Value: int32(binary.LittleEndian.Uint32(buf[20:24])),
			}

			if event.Type != EV_KEY {
				continue
			}

			key := translateLinuxKeycode(uint32(event.Code))
			if key == KeyUnknown {
				continue
			}

			var evt KeyEvent
			evt.RawCode = int64(event.Code)
			evt.Keycode = key

			switch event.Value {
			case KEY_PRESSED:
				if key == KeyToggle {
					evt.EventType = FlagsChanged
					h.eventChan <- evt
				} else if mc != nil && mc.IsActive() {
					evt.EventType = KeyDown
					h.eventChan <- evt
				}
			case KEY_RELEASED:
				if mc != nil && mc.IsActive() && key != KeyToggle {
					evt.EventType = KeyUp
					h.eventChan <- evt
				}
			}
		}
	}()

	return h.eventChan, nil
}

func (h *LinuxKeyboardHook) Stop() error {
	h.running = false
	if h.device != nil {
		h.device.Close()
	}
	close(h.eventChan)
	return nil
}

// translateLinuxKeycode converts Linux evdev code to unified Key
func translateLinuxKeycode(code uint32) Key {
	switch code {
	case linuxKeyCapsLock:
		return KeyToggle
	case linuxKeyW:
		return KeyMoveUp
	case linuxKeyS:
		return KeyMoveDown
	case linuxKeyA:
		return KeyMoveLeft
	case linuxKeyD:
		return KeyMoveRight
	case linuxKeyQ:
		return KeyDiagUpLeft
	case linuxKeyE:
		return KeyDiagUpRight
	case linuxKeyZ:
		return KeyDiagDownLeft
	case linuxKeyX:
		return KeyDiagDownRight
	case linuxKeySpace:
		return KeyLeftClick
	case linuxKeyLeftCtrl:
		return KeyRightClick
	case linuxKeyLeftShift:
		return KeyMiddleClick
	case linuxKeyR:
		return KeyScrollUp
	case linuxKeyF:
		return KeyScrollDown
	default:
		return KeyUnknown
	}
}

// findKeyboardDevice finds the first keyboard device in /dev/input
func findKeyboardDevice() (string, error) {
	// Try to find a keyboard in /dev/input/by-id
	byIdPath := "/dev/input/by-id"
	entries, err := os.ReadDir(byIdPath)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.Contains(name, "kbd") || strings.Contains(name, "keyboard") {
				return filepath.Join(byIdPath, name), nil
			}
		}
	}

	// Fallback: check /proc/bus/input/devices
	devicesFile, err := os.Open("/proc/bus/input/devices")
	if err != nil {
		return "", err
	}
	defer devicesFile.Close()

	scanner := bufio.NewScanner(devicesFile)
	var currentHandler string
	isKeyboard := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "N: Name=") {
			name := strings.ToLower(line)
			isKeyboard = strings.Contains(name, "keyboard") || strings.Contains(name, "kbd")
		}

		if strings.HasPrefix(line, "H: Handlers=") && isKeyboard {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event") {
					currentHandler = part
					return "/dev/input/" + currentHandler, nil
				}
			}
		}

		if line == "" {
			isKeyboard = false
			currentHandler = ""
		}
	}

	// Last resort: try event0
	return "/dev/input/event0", nil
}
