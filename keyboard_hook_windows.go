//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procSetWindowsHookEx = user32.NewProc("SetWindowsHookExW")
	procCallNextHookEx   = user32.NewProc("CallNextHookEx")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procGetMessage       = user32.NewProc("GetMessageW")
)

const (
	WH_KEYBOARD_LL = 13
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105
)

// Windows Virtual Key codes
const (
	VK_CAPITAL   = 0x14 // Caps Lock
	VK_W         = 0x57
	VK_A         = 0x41
	VK_S         = 0x53
	VK_D         = 0x44
	VK_Q         = 0x51
	VK_E         = 0x45
	VK_Z         = 0x5A
	VK_X         = 0x58
	VK_R         = 0x52
	VK_F         = 0x46
	VK_SPACE     = 0x20
	VK_LCONTROL  = 0xA2
	VK_LSHIFT    = 0xA0
)

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

// Global variables for hook
var (
	windowsEventChan chan KeyEvent
	windowsHookHandle uintptr
)

// WindowsKeyboardHook implements KeyboardHook for Windows
type WindowsKeyboardHook struct {
	eventChan chan KeyEvent
	running   bool
}

// NewKeyboardHook creates a new keyboard hook for Windows
func NewKeyboardHook() KeyboardHook {
	return &WindowsKeyboardHook{
		eventChan: make(chan KeyEvent, 100),
	}
}

func (h *WindowsKeyboardHook) Start() (<-chan KeyEvent, error) {
	windowsEventChan = h.eventChan
	h.running = true

	go func() {
		// Set up the low-level keyboard hook
		hookProc := syscall.NewCallback(keyboardProc)
		handle, _, _ := procSetWindowsHookEx.Call(
			WH_KEYBOARD_LL,
			hookProc,
			0,
			0,
		)
		windowsHookHandle = handle

		// Message loop
		var msg MSG
		for {
			ret, _, _ := procGetMessage.Call(
				uintptr(unsafe.Pointer(&msg)),
				0,
				0,
				0,
			)
			if ret == 0 {
				break
			}
		}
	}()

	return h.eventChan, nil
}

func (h *WindowsKeyboardHook) Stop() error {
	if windowsHookHandle != 0 {
		procUnhookWindowsHookEx.Call(windowsHookHandle)
		windowsHookHandle = 0
	}
	h.running = false
	close(h.eventChan)
	return nil
}

// translateWindowsKeycode converts Windows VK code to unified Key
func translateWindowsKeycode(vkCode uint32) Key {
	switch vkCode {
	case VK_CAPITAL:
		return KeyToggle
	case VK_W:
		return KeyMoveUp
	case VK_S:
		return KeyMoveDown
	case VK_A:
		return KeyMoveLeft
	case VK_D:
		return KeyMoveRight
	case VK_Q:
		return KeyDiagUpLeft
	case VK_E:
		return KeyDiagUpRight
	case VK_Z:
		return KeyDiagDownLeft
	case VK_X:
		return KeyDiagDownRight
	case VK_SPACE:
		return KeyLeftClick
	case VK_LCONTROL:
		return KeyRightClick
	case VK_LSHIFT:
		return KeyMiddleClick
	case VK_R:
		return KeyScrollUp
	case VK_F:
		return KeyScrollDown
	default:
		return KeyUnknown
	}
}

// keyboardProc is the callback for the Windows keyboard hook
func keyboardProc(nCode int, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 {
		kbStruct := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		key := translateWindowsKeycode(kbStruct.VkCode)

		var evt KeyEvent
		evt.RawCode = int64(kbStruct.VkCode)
		evt.Keycode = key

		switch wParam {
		case WM_KEYDOWN, WM_SYSKEYDOWN:
			if key == KeyToggle {
				evt.EventType = FlagsChanged
				if windowsEventChan != nil {
					windowsEventChan <- evt
				}
			} else if mc != nil && mc.IsActive() && key != KeyUnknown {
				evt.EventType = KeyDown
				if windowsEventChan != nil {
					windowsEventChan <- evt
				}
				// Return 1 to suppress the key
				return 1
			}
		case WM_KEYUP, WM_SYSKEYUP:
			if mc != nil && mc.IsActive() && key != KeyUnknown && key != KeyToggle {
				evt.EventType = KeyUp
				if windowsEventChan != nil {
					windowsEventChan <- evt
				}
				// Return 1 to suppress the key
				return 1
			}
		}
	}

	// Call next hook in the chain
	ret, _, _ := procCallNextHookEx.Call(0, uintptr(nCode), wParam, lParam)
	return ret
}
