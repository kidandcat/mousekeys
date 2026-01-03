//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// LinuxAutostart implements Autostart for Linux using XDG autostart
type LinuxAutostart struct{}

// NewAutostart creates a new autostart handler for Linux
func NewAutostart() Autostart {
	return &LinuxAutostart{}
}

func (a *LinuxAutostart) getAutostartDir() string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, _ := os.UserHomeDir()
		config = filepath.Join(home, ".config")
	}
	return filepath.Join(config, "autostart")
}

func (a *LinuxAutostart) getDesktopFilePath() string {
	return filepath.Join(a.getAutostartDir(), "mousekeys.desktop")
}

func (a *LinuxAutostart) IsEnabled() bool {
	_, err := os.Stat(a.getDesktopFilePath())
	return err == nil
}

func (a *LinuxAutostart) Enable() error {
	// Create autostart directory if it doesn't exist
	autostartDir := a.getAutostartDir()
	if err := os.MkdirAll(autostartDir, 0755); err != nil {
		return err
	}

	exe, _ := os.Executable()

	desktopEntry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=MouseKeys
Comment=Keyboard-based mouse control
Exec=%s
Icon=input-mouse
Terminal=false
Categories=Utility;Accessibility;
X-GNOME-Autostart-enabled=true
`, exe)

	return os.WriteFile(a.getDesktopFilePath(), []byte(desktopEntry), 0644)
}

func (a *LinuxAutostart) Disable() error {
	return os.Remove(a.getDesktopFilePath())
}
