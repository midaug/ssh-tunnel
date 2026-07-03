package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"ssh-tunnel/internal/autostart"
	"ssh-tunnel/internal/config"
	"ssh-tunnel/internal/dock"
	"ssh-tunnel/internal/tunnel"
)

// App 主应用，绑定到前端
type App struct {
	ctx        context.Context
	store      *config.Store
	manager    *tunnel.Manager
	refreshTray func()
	quitAppFn  func()
}

func NewApp(store *config.Store, manager *tunnel.Manager) *App {
	return &App{store: store, manager: manager}
}

// setRefreshTray 注入托盘刷新回调（包内使用，不导出给前端）
func (a *App) setRefreshTray(fn func()) { a.refreshTray = fn }

// setQuitApp 注入完全退出回调
func (a *App) setQuitApp(fn func()) { a.quitAppFn = fn }

func (a *App) refresh() {
	if a.refreshTray != nil {
		a.refreshTray()
	}
}

// quitApp 完全退出程序
func (a *App) quitApp() {
	if a.quitAppFn != nil {
		a.quitAppFn()
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// --- 隧道管理 ---

// TunnelList 返回所有隧道配置（含运行时状态）
func (a *App) TunnelList() []config.Tunnel {
	return a.store.ListTunnels()
}

// TunnelSave 新增或更新隧道
func (a *App) TunnelSave(t config.Tunnel) (config.Tunnel, error) {
	saved, err := a.store.SaveTunnel(t)
	if err != nil {
		return config.Tunnel{}, err
	}
	// 若隧道正在运行，重启以应用新配置
	if a.manager.IsRunning(saved.ID) {
		_ = a.manager.Restart(saved.ID)
	} else {
		a.manager.ReloadConfig(saved.ID)
	}
	a.refresh()
	return saved, nil
}

// TunnelDelete 删除隧道（运行中则先停止）
func (a *App) TunnelDelete(id string) error {
	_ = a.manager.Stop(id)
	err := a.store.DeleteTunnel(id)
	a.refresh()
	return err
}

// TunnelStart 启动隧道
func (a *App) TunnelStart(id string) error {
	return a.manager.Start(id)
}

// TunnelStop 停止隧道
func (a *App) TunnelStop(id string) error {
	return a.manager.Stop(id)
}

// TunnelRestart 重启隧道
func (a *App) TunnelRestart(id string) error {
	return a.manager.Restart(id)
}

// TunnelStartAll 启动所有隧道
func (a *App) TunnelStartAll() {
	a.manager.StartAll()
}

// TunnelStopAll 停止所有隧道
func (a *App) TunnelStopAll() {
	a.manager.StopAll()
}

// --- 配置导入导出 ---

// ExportConfig 导出配置为 JSON 字符串
func (a *App) ExportConfig() (string, error) {
	return a.store.ExportConfig()
}

// ImportConfig 导入配置 JSON，mode: "merge" | "replace"
func (a *App) ImportConfig(jsonStr string, mode string) (int, error) {
	if mode == "" {
		mode = "merge"
	}
	return a.store.ImportConfig(jsonStr, mode)
}

// ParseSSHCommand 解析 ssh 命令行为隧道配置
func (a *App) ParseSSHCommand(cmdline string) (config.Tunnel, error) {
	return config.ParseSSHCommand(cmdline)
}

// ToSSHCommand 把隧道配置导出为 ssh 命令行
func (a *App) ToSSHCommand(t config.Tunnel) string {
	return config.ToSSHCommand(t)
}

// ToSSHCommandByID 按 ID 导出某条隧道的 ssh 命令
func (a *App) ToSSHCommandByID(id string) (string, error) {
	t, err := a.store.GetTunnel(id)
	if err != nil {
		return "", err
	}
	return config.ToSSHCommand(t), nil
}

// --- 测试连接 ---

// TestConnection 仅测试 SSH 连通性，不启动转发
func (a *App) TestConnection(t config.Tunnel) error {
	return tunnel.TestConnection(t)
}

// --- 设置 ---

// GetSettings 返回全局设置
func (a *App) GetSettings() config.Settings {
	return a.store.GetSettings()
}

// SetSettings 更新全局设置（同时同步开机自启、Dock 显示）
func (a *App) SetSettings(s config.Settings) error {
	if s.Autostart {
		if err := autostart.Enable(""); err != nil {
			return fmt.Errorf("启用开机自启失败: %w", err)
		}
	} else {
		if err := autostart.Disable(); err != nil {
			return fmt.Errorf("关闭开机自启失败: %w", err)
		}
	}
	// 实时切换 Dock 显示
	if s.HideFromDock {
		dock.Hide()
	} else {
		dock.Show()
	}
	return a.store.SetSettings(s)
}

// GetAutostart 返回系统层面开机自启是否启用
func (a *App) GetAutostart() bool {
	return autostart.Enabled()
}

// QuitApp 完全退出程序（停所有隧道 + 退托盘 + 退 Wails）
func (a *App) QuitApp() {
	a.quitApp()
}

// --- 工具 ---

// ConfigPath 返回配置文件路径
func (a *App) ConfigPath() string {
	return a.store.Path()
}

// OpenInFolder 在文件管理器中显示文件
func (a *App) OpenInFolder(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	case "windows":
		return exec.Command("explorer", "/select,", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}
