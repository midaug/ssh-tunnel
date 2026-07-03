//go:build windows

package autostart

import (
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// Enabled 返回是否启用开机自启
func Enabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(AppName)
	return err == nil
}

// Enable 启用开机自启
func Enable(execPath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(AppName, `"`+execPath+`"`)
}

// Disable 关闭开机自启
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	err = k.DeleteValue(AppName)
	// 值不存在视为成功（未启用过自启时删除会返回 FILE_NOT_FOUND）
	if err != nil && (err == registry.ErrNotExist || err == windows.ERROR_FILE_NOT_FOUND || os.IsNotExist(err)) {
		return nil
	}
	return err
}
