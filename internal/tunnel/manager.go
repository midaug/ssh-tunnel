package tunnel

import (
	"sync"

	"ssh-tunnel/internal/config"
)

// Manager 管理所有隧道的运行时实例
type Manager struct {
	mu       sync.RWMutex
	tunnels  map[string]*Tunnel
	store    *config.Store
	onStatus EventHandler
}

// NewManager 创建管理器。onStatus 用于向 Wails 前端推送状态事件。
func NewManager(store *config.Store, onStatus EventHandler) *Manager {
	return &Manager{
		tunnels:  make(map[string]*Tunnel),
		store:    store,
		onStatus: onStatus,
	}
}

// ensure 确保隧道实例存在（按当前 store 中的配置创建）
func (m *Manager) ensure(id string) (*Tunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tunnels[id]; ok {
		return t, nil
	}
	cfg, err := m.store.GetTunnel(id)
	if err != nil {
		return nil, err
	}
	t := New(cfg, m.onStatus)
	m.tunnels[id] = t
	return t, nil
}

// Start 启动指定隧道
func (m *Manager) Start(id string) error {
	t, err := m.ensure(id)
	if err != nil {
		return err
	}
	// 同步最新配置
	cfg, _ := m.store.GetTunnel(id)
	t.mu.Lock()
	t.cfg = cfg
	t.mu.Unlock()
	return t.Start()
}

// Stop 停止指定隧道
func (m *Manager) Stop(id string) error {
	m.mu.RLock()
	t, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	t.Stop()
	return nil
}

// Restart 重启
func (m *Manager) Restart(id string) error {
	_ = m.Stop(id)
	return m.Start(id)
}

// ReloadConfig 配置变更后同步到运行中的 Tunnel（不重启，下次重连生效）
func (m *Manager) ReloadConfig(id string) {
	m.mu.RLock()
	t, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	cfg, err := m.store.GetTunnel(id)
	if err != nil {
		return
	}
	t.mu.Lock()
	t.cfg = cfg
	t.mu.Unlock()
}

// Status 返回状态
func (m *Manager) Status(id string) (config.Status, string) {
	m.mu.RLock()
	t, ok := m.tunnels[id]
	m.mu.RUnlock()
	if !ok {
		return config.StatusStopped, ""
	}
	s, e, _ := t.Status()
	return s, e
}

// StopAll 停止所有隧道（退出时调用）
func (m *Manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.tunnels))
	for id := range m.tunnels {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	for _, id := range ids {
		m.Stop(id)
	}
}

// StartAll 启动所有隧道
func (m *Manager) StartAll() {
	ids := m.store.ListTunnelIDs()
	for _, id := range ids {
		_ = m.Start(id)
	}
}

// IsRunning 检查是否在运行
func (m *Manager) IsRunning(id string) bool {
	s, _ := m.Status(id)
	return s == config.StatusRunning || s == config.StatusConnecting
}
