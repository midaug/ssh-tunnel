package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const appDirName = "ssh-tunnel"
const configFile = "config.json"

var (
	ErrTunnelNotFound = errors.New("tunnel not found")
)

func newID() string { return uuid.NewString() }

// Store 配置持久化，JSON 文件读写
type Store struct {
	mu   sync.RWMutex
	path string
	cfg  *Config
}

// NewStore 创建存储并加载（不存在则创建默认配置）
func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(dir, appDirName)
	if err := os.MkdirAll(appDir, 0o700); err != nil {
		return nil, err
	}
	s := &Store{
		path: filepath.Join(appDir, configFile),
		cfg: &Config{
			Version: CurrentVersion,
			Settings: Settings{
				MinimizeToTray: true,
			},
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// Path 返回配置文件路径
func (s *Store) Path() string { return s.path }

// Dir 返回配置目录
func (s *Store) Dir() string { return filepath.Dir(s.path) }

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.persist()
	}
	if err != nil {
		return err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return err
	}
	if c.Version == "" {
		c.Version = CurrentVersion
	}
	// 重置运行时状态：上次退出时残留的 running/connecting 状态已失效
	for i := range c.Tunnels {
		c.Tunnels[i].Status = StatusStopped
		c.Tunnels[i].LastError = ""
		c.Tunnels[i].StartedAt = time.Time{}
	}
	s.cfg = &c
	return s.persist()
}

func (s *Store) persist() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s.cfg, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// Get 返回配置快照（深拷贝）
func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return *s.cfg
}

// GetSettings 返回设置
func (s *Store) GetSettings() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Settings
}

// SetSettings 更新设置并持久化
func (s *Store) SetSettings(set Settings) error {
	s.mu.Lock()
	s.cfg.Settings = set
	s.mu.Unlock()
	return s.persist()
}

// ListTunnels 返回隧道列表
func (s *Store) ListTunnels() []Tunnel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Tunnel, len(s.cfg.Tunnels))
	copy(out, s.cfg.Tunnels)
	return out
}

// ListTunnelIDs 返回所有隧道 ID
func (s *Store) ListTunnelIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, 0, len(s.cfg.Tunnels))
	for _, t := range s.cfg.Tunnels {
		out = append(out, t.ID)
	}
	return out
}

// SaveTunnel 新增或更新（按 ID），返回保存后的 Tunnel
func (s *Store) SaveTunnel(t Tunnel) (Tunnel, error) {
	if t.ID == "" {
		t.ID = newID()
	}
	s.mu.Lock()
	found := false
	for i, x := range s.cfg.Tunnels {
		if x.ID == t.ID {
			t.Status = x.Status
			t.StartedAt = x.StartedAt
			t.LastError = x.LastError
			s.cfg.Tunnels[i] = t
			found = true
			break
		}
	}
	if !found {
		t.Status = StatusStopped
		s.cfg.Tunnels = append(s.cfg.Tunnels, t)
	}
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		return t, err
	}
	return t, nil
}

// DeleteTunnel 删除
func (s *Store) DeleteTunnel(id string) error {
	s.mu.Lock()
	idx := -1
	for i, x := range s.cfg.Tunnels {
		if x.ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return ErrTunnelNotFound
	}
	s.cfg.Tunnels = append(s.cfg.Tunnels[:idx], s.cfg.Tunnels[idx+1:]...)
	s.mu.Unlock()
	return s.persist()
}

// GetTunnel 查找
func (s *Store) GetTunnel(id string) (Tunnel, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, x := range s.cfg.Tunnels {
		if x.ID == id {
			return x, nil
		}
	}
	return Tunnel{}, ErrTunnelNotFound
}

// UpdateRuntimeStatus 仅更新运行时状态，不持久化到磁盘（避免频繁写盘）
func (s *Store) UpdateRuntimeStatus(id string, status Status, lastErr string, startedAtZero bool) {
	s.mu.Lock()
	for i := range s.cfg.Tunnels {
		if s.cfg.Tunnels[i].ID == id {
			s.cfg.Tunnels[i].Status = status
			s.cfg.Tunnels[i].LastError = lastErr
			if startedAtZero {
				s.cfg.Tunnels[i].StartedAt = time.Time{}
			} else if status == StatusRunning || status == StatusConnecting {
				s.cfg.Tunnels[i].StartedAt = time.Now()
			}
			break
		}
	}
	s.mu.Unlock()
}
