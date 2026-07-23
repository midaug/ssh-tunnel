package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"ssh-tunnel/internal/auth"
	"ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// EventHandler 隧道状态变更回调（用于向 Wails 前端推送事件）
type EventHandler func(tunnelID string, status config.Status, lastError string)

// Tunnel 单条隧道的运行时实例
type Tunnel struct {
	cfg      config.Tunnel
	mu       sync.Mutex
	client   *ssh.Client
	stopCh   chan struct{}
	doneCh   chan struct{}
	listeners []net.Listener
	status   config.Status
	lastErr  string
	startedAt time.Time

	onStatus EventHandler
}

// New 用配置创建一个 Tunnel 实例（不启动）
func New(c config.Tunnel, h EventHandler) *Tunnel {
	return &Tunnel{cfg: c, onStatus: h}
}

// Cfg 返回当前配置快照
func (t *Tunnel) Cfg() config.Tunnel {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cfg
}

// Status 返回运行时状态
func (t *Tunnel) Status() (config.Status, string, time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.lastErr, t.startedAt
}

func (t *Tunnel) setStatus(s config.Status, err string) {
	t.mu.Lock()
	t.status = s
	if err != "" || s == config.StatusRunning || s == config.StatusConnecting {
		t.lastErr = err
	}
	if s == config.StatusRunning || s == config.StatusConnecting {
		if t.startedAt.IsZero() {
			t.startedAt = time.Now()
		}
	} else {
		t.startedAt = time.Time{}
	}
	statusCopy := s
	errCopy := err
	h := t.onStatus
	t.mu.Unlock()
	if h != nil {
		h(t.cfg.ID, statusCopy, errCopy)
	}
}

// Start 启动隧道。若已运行则直接返回。自动重连 goroutine 在断线后重试。
func (t *Tunnel) Start() error {
	t.mu.Lock()
	if t.status == config.StatusRunning || t.status == config.StatusConnecting {
		t.mu.Unlock()
		return nil
	}
	if t.stopCh != nil {
		t.mu.Unlock()
		return errors.New("隧道正在停止中")
	}
	t.stopCh = make(chan struct{})
	t.doneCh = make(chan struct{})
	t.mu.Unlock()

	go t.run()
	return nil
}

// Stop 停止隧道，阻塞直到 goroutine 退出
func (t *Tunnel) Stop() {
	t.mu.Lock()
	if t.stopCh == nil {
		t.mu.Unlock()
		return
	}
	select {
	case <-t.stopCh:
	default:
		close(t.stopCh)
	}
	done := t.doneCh
	t.mu.Unlock()
	if done != nil {
		<-done
	}
}

// run 主循环：连接 → 启动转发 → 监控断线 → 退避重连
func (t *Tunnel) run() {
	defer func() {
		t.setStatus(config.StatusStopped, "")
		t.mu.Lock()
		if t.doneCh != nil {
			close(t.doneCh)
			t.doneCh = nil
			t.stopCh = nil
		}
		t.mu.Unlock()
	}()

	backoff := t.cfg.ReconnectMinMS
	if backoff < 500 {
		backoff = 500
	}
	maxBackoff := t.cfg.ReconnectMaxMS
	if maxBackoff < backoff {
		maxBackoff = 30000
	}

	for {
		select {
		case <-t.stopCh:
			return
		default:
		}

		t.setStatus(config.StatusConnecting, "")
		client, err := t.connect()
		if err != nil {
			// 若是停止导致的中断，不报错
			select {
			case <-t.stopCh:
				return
			default:
			}
			t.setStatus(config.StatusError, err.Error())
			if !t.cfg.AutoReconnect {
				return
			}
			if !t.sleep(time.Duration(backoff) * time.Millisecond) {
				return
			}
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		t.mu.Lock()
		t.client = client
		t.mu.Unlock()

		// 启动所有转发监听器
		fwdErr := t.startForwards(client)
		if fwdErr != nil {
			t.setStatus(config.StatusError, fwdErr.Error())
			t.closeAll()
			_ = client.Close()
			if !t.cfg.AutoReconnect {
				return
			}
			if !t.sleep(time.Duration(backoff) * time.Millisecond) {
				return
			}
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// 重连成功，重置退避
		backoff = t.cfg.ReconnectMinMS
		if backoff < 500 {
			backoff = 500
		}
		t.setStatus(config.StatusRunning, "")

		// 阻塞等待断线或停止
		waitErr := make(chan error, 1)
		go func() { waitErr <- client.Wait() }()
		select {
		case <-t.stopCh:
			t.closeAll()
			_ = client.Close()
			return
		case <-waitErr:
			t.closeAll()
			_ = client.Close()
			t.mu.Lock()
			t.client = nil
			t.mu.Unlock()
			if !t.cfg.AutoReconnect {
				t.setStatus(config.StatusError, "连接断开")
				return
			}
			t.setStatus(config.StatusConnecting, "连接断开，准备重连")
			if !t.sleep(time.Duration(backoff) * time.Millisecond) {
				return
			}
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

// connect 建立 SSH 连接。使用 context 监听 stopCh，使 Stop 时能立即中断拨号
func (t *Tunnel) connect() (*ssh.Client, error) {
	authMethods, err := auth.BuildAuthMethods(string(t.cfg.AuthType), t.cfg.Password, t.cfg.KeyPath, t.cfg.KeyPassphrase)
	if err != nil {
		return nil, fmt.Errorf("认证配置错误: %w", err)
	}
	port := t.cfg.Port
	if port == 0 {
		port = 22
	}
	sshCfg := &ssh.ClientConfig{
		User:            t.cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 简化：不校验 host key，管理工具常见做法
		Timeout:         15 * time.Second,
	}
	if t.cfg.ServerAliveInterval > 0 {
		sshCfg.Config.SetDefaults()
	}
	addr := fmt.Sprintf("%s:%d", t.cfg.Host, port)

	// 用 context 让 TCP 拨号可被 stopCh 中断
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stopWatcherDone := make(chan struct{})
	go func() {
		select {
		case <-t.stopCh:
			cancel()
		case <-stopWatcherDone:
		}
	}()
	defer close(stopWatcherDone)

	conn, err := dialViaProxy(ctx, t.cfg.Proxy, addr)
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("已取消")
		}
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}
	// 握手阶段若被取消则关闭连接
	sc, chans, reqs, err := ssh.NewClientConn(conn, addr, sshCfg)
	if err != nil {
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil, errors.New("已取消")
		}
		return nil, fmt.Errorf("握手 %s 失败: %w", addr, err)
	}
	return ssh.NewClient(sc, chans, reqs), nil
}

// startForwards 启动所有转发监听器
func (t *Tunnel) startForwards(client *ssh.Client) error {
	t.mu.Lock()
	t.listeners = t.listeners[:0]
	t.mu.Unlock()
	for _, f := range t.cfg.Forwards {
		if err := t.startOneForward(client, f); err != nil {
			return fmt.Errorf("转发 %s %s 启动失败: %w", f.Type, f.Listen, err)
		}
	}
	return nil
}

// closeAll 关闭所有监听器
func (t *Tunnel) closeAll() {
	t.mu.Lock()
	ls := t.listeners
	t.listeners = nil
	t.mu.Unlock()
	for _, l := range ls {
		_ = l.Close()
	}
}

// sleep 可被停止信号打断的 sleep
func (t *Tunnel) sleep(d time.Duration) bool {
	if d <= 0 {
		return true
	}
	select {
	case <-t.stopCh:
		return false
	case <-time.After(d):
		return true
	}
}

// normalizeAddr 规范化监听地址，缺少 host 时默认 127.0.0.1
func normalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(addr); err != nil {
		// 只有端口
		return "127.0.0.1:" + addr
	}
	return addr
}

// hostPort 拆出 host 和 port
func hostPort(addr string) (string, string) {
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		return "", ""
	}
	return h, p
}
