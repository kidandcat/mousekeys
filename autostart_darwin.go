//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DarwinAutostart implements Autostart for macOS using LaunchAgent
type DarwinAutostart struct{}

// NewAutostart creates a new autostart handler for macOS
func NewAutostart() Autostart {
	return &DarwinAutostart{}
}

func (a *DarwinAutostart) getLaunchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.mousekeys.app.plist")
}

func (a *DarwinAutostart) getAppPath() string {
	exe, _ := os.Executable()
	// If running from .app bundle, return the .app path
	// exe will be like /Applications/MouseKeys.app/Contents/MacOS/mousekeys
	if idx := strings.Index(exe, ".app/"); idx != -1 {
		return exe[:idx+4] // Include ".app"
	}
	return exe
}

func (a *DarwinAutostart) IsEnabled() bool {
	_, err := os.Stat(a.getLaunchAgentPath())
	return err == nil
}

func (a *DarwinAutostart) Enable() error {
	appPath := a.getAppPath()
	var plist string

	if strings.HasSuffix(appPath, ".app") {
		// Use open command for .app bundles
		plist = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mousekeys.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/open</string>
        <string>-a</string>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, appPath)
	} else {
		// Direct binary execution
		plist = fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.mousekeys.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>`, appPath)
	}

	return os.WriteFile(a.getLaunchAgentPath(), []byte(plist), 0644)
}

func (a *DarwinAutostart) Disable() error {
	return os.Remove(a.getLaunchAgentPath())
}
