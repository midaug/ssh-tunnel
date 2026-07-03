//go:build darwin

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const launchAgentID = "com.ssh-tunnel.app"

func plistPath() string {
	return filepath.Join(homeDir(), "Library", "LaunchAgents", launchAgentID+".plist")
}

// Enabled 返回当前是否启用开机自启
func Enabled() bool {
	_, err := os.Stat(plistPath())
	return err == nil
}

// Enable 启用开机自启
func Enable(execPath string) error {
	if err := ensureDir(plistPath()); err != nil {
		return err
	}
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <false/>
</dict>
</plist>
`, launchAgentID, execPath)
	return os.WriteFile(plistPath(), []byte(plist), 0o644)
}

// Disable 关闭开机自启
func Disable() error {
	err := os.Remove(plistPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
