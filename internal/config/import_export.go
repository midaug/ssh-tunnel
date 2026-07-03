package config

import (
	"encoding/json"
	"os"
	"time"
)

// ExportConfig 导出整个配置为 JSON 字符串（不含运行时状态，便于分享）
func (s *Store) ExportConfig() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Config{
		Version:  s.cfg.Version,
		Settings: s.cfg.Settings,
	}
	for _, t := range s.cfg.Tunnels {
		t.Status = ""
		t.LastError = ""
		t.StartedAt = time.Time{}
		out.Tunnels = append(out.Tunnels, t)
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportConfig 从 JSON 字符串导入，mode: "merge" 合并 | "replace" 覆盖
// 返回导入的隧道数量
func (s *Store) ImportConfig(jsonStr string, mode string) (int, error) {
	var in Config
	if err := json.Unmarshal([]byte(jsonStr), &in); err != nil {
		return 0, err
	}
	if in.Version == "" {
		in.Version = CurrentVersion
	}
	s.mu.Lock()
	if mode == "replace" {
		s.cfg.Tunnels = in.Tunnels
		s.cfg.Settings = in.Settings
	} else {
		// merge：按 ID 去重
		exist := map[string]bool{}
		for _, t := range s.cfg.Tunnels {
			exist[t.ID] = true
		}
		for _, t := range in.Tunnels {
			if t.ID == "" {
				t.ID = newID()
			}
			if exist[t.ID] {
				// 重命名避免覆盖
				t.ID = newID()
			}
			t.Status = StatusStopped
			s.cfg.Tunnels = append(s.cfg.Tunnels, t)
			exist[t.ID] = true
		}
	}
	s.mu.Unlock()
	if err := s.persist(); err != nil {
		return 0, err
	}
	return len(in.Tunnels), nil
}

// ExportToFile 导出到文件
func (s *Store) ExportToFile(path string) error {
	data, err := s.ExportConfig()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o600)
}
