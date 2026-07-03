package autostart

import (
	"os"
	"path/filepath"
)

// AppName 注册项名称
const AppName = "SSH Tunnel"

// configDir 返回用户配置目录
func configDir() string {
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return d
}

// homeDir 返回家目录
func homeDir() string {
	d, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return d
}

// ensureDir 确保目录存在
func ensureDir(p string) error {
	return os.MkdirAll(filepath.Dir(p), 0o755)
}
