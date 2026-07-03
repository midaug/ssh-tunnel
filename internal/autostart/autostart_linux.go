//go:build linux

package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopFileName = "ssh-tunnel.desktop"

func desktopPath() string {
	return filepath.Join(homeDir(), ".config", "autostart", desktopFileName)
}

// Enabled 返回当前是否启用开机自启
func Enabled() bool {
	_, err := os.Stat(desktopPath())
	return err == nil
}

// Enable 启用开机自启
func Enable(execPath string) error {
	if err := ensureDir(desktopPath()); err != nil {
		return err
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, AppName, execPath)
	return os.WriteFile(desktopPath(), []byte(content), 0o644)
}

// Disable 关闭开机自启
func Disable() error {
	err := os.Remove(desktopPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
