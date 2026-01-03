//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	advapi32           = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyEx   = advapi32.NewProc("RegOpenKeyExW")
	procRegCloseKey    = advapi32.NewProc("RegCloseKey")
	procRegSetValueEx  = advapi32.NewProc("RegSetValueExW")
	procRegDeleteValue = advapi32.NewProc("RegDeleteValueW")
	procRegQueryValueEx = advapi32.NewProc("RegQueryValueExW")
)

const (
	HKEY_CURRENT_USER = 0x80000001
	KEY_READ          = 0x20019
	KEY_WRITE         = 0x20006
	REG_SZ            = 1
)

// WindowsAutostart implements Autostart for Windows using Registry
type WindowsAutostart struct{}

// NewAutostart creates a new autostart handler for Windows
func NewAutostart() Autostart {
	return &WindowsAutostart{}
}

func (a *WindowsAutostart) getRegistryPath() string {
	return `Software\Microsoft\Windows\CurrentVersion\Run`
}

func (a *WindowsAutostart) IsEnabled() bool {
	keyPath, _ := syscall.UTF16PtrFromString(a.getRegistryPath())
	valueName, _ := syscall.UTF16PtrFromString("MouseKeys")

	var hKey uintptr
	ret, _, _ := procRegOpenKeyEx.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		KEY_READ,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return false
	}
	defer procRegCloseKey.Call(hKey)

	ret, _, _ = procRegQueryValueEx.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		0,
		0,
		0,
	)
	return ret == 0
}

func (a *WindowsAutostart) Enable() error {
	keyPath, _ := syscall.UTF16PtrFromString(a.getRegistryPath())
	valueName, _ := syscall.UTF16PtrFromString("MouseKeys")

	exe, _ := os.Executable()
	exePath, _ := syscall.UTF16FromString(exe)

	var hKey uintptr
	ret, _, _ := procRegOpenKeyEx.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		KEY_WRITE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	defer procRegCloseKey.Call(hKey)

	ret, _, _ = procRegSetValueEx.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
		0,
		REG_SZ,
		uintptr(unsafe.Pointer(&exePath[0])),
		uintptr(len(exePath)*2),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}

	return nil
}

func (a *WindowsAutostart) Disable() error {
	keyPath, _ := syscall.UTF16PtrFromString(a.getRegistryPath())
	valueName, _ := syscall.UTF16PtrFromString("MouseKeys")

	var hKey uintptr
	ret, _, _ := procRegOpenKeyEx.Call(
		HKEY_CURRENT_USER,
		uintptr(unsafe.Pointer(keyPath)),
		0,
		KEY_WRITE,
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}
	defer procRegCloseKey.Call(hKey)

	ret, _, _ = procRegDeleteValue.Call(
		hKey,
		uintptr(unsafe.Pointer(valueName)),
	)
	if ret != 0 {
		return syscall.Errno(ret)
	}

	return nil
}
