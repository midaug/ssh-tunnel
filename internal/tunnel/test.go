package tunnel

import (
	"context"
	"fmt"
	"time"

	"ssh-tunnel/internal/auth"
	"ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// TestConnection 仅测试 SSH 连通性，不启动任何转发。
// 若配置了代理，则连接经代理建立。
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	conn, err := dialViaProxy(ctx, c.Proxy, addr)
	if err != nil {
		return fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	// 握手阶段设置超时，避免代理连上但对端不是 SSH 时长期阻塞
	_ = conn.SetDeadline(time.Now().Add(cfg.Timeout))
	sc, chans, reqs, err := ssh.NewClientConn(conn, addr, cfg)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("握手 %s 失败: %w", addr, err)
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(sc, chans, reqs)
	_ = client.Close()
	return nil
}
