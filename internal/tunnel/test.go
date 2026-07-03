package tunnel

import (
	"fmt"
	"time"

	"ssh-tunnel/internal/auth"
	"ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// TestConnection 仅测试 SSH 连通性，不启动任何转发
func TestConnection(c config.Tunnel) error {
	authMethods, err := auth.BuildAuthMethods(string(c.AuthType), c.Password, c.KeyPath, c.KeyPassphrase)
	if err != nil {
		return fmt.Errorf("认证配置错误: %w", err)
	}
	port := c.Port
	if port == 0 {
		port = 22
	}
	cfg := &ssh.ClientConfig{
		User:            c.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", c.Host, port)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	_ = client.Close()
	return nil
}
