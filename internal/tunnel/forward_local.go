package tunnel

import (
	"fmt"
	"io"
	"net"

	"ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// startOneForward 根据类型启动单个转发
func (t *Tunnel) startOneForward(client *ssh.Client, f config.Forward) error {
	switch f.Type {
	case config.ForwardLocal:
		return t.startLocal(client, f)
	case config.ForwardRemote:
		return t.startRemote(client, f)
	case config.ForwardDynamic:
		return t.startDynamic(client, f)
	default:
		return fmt.Errorf("未知转发类型: %s", f.Type)
	}
}

// startLocal 本地转发 -L listen:targetHost:targetPort
// 在本地监听 listen，每个连接通过 ssh 拨号到 target
func (t *Tunnel) startLocal(client *ssh.Client, f config.Forward) error {
	listen := normalizeAddr(f.Listen)
	if listen == "" {
		return fmt.Errorf("监听地址为空")
	}
	target := f.Target
	if target == "" {
		return fmt.Errorf("目标地址为空")
	}
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("本地监听 %s 失败: %w", listen, err)
	}
	t.mu.Lock()
	t.listeners = append(t.listeners, ln)
	t.mu.Unlock()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go t.handleLocalConn(client, conn, target)
		}
	}()
	return nil
}

func (t *Tunnel) handleLocalConn(client *ssh.Client, local net.Conn, target string) {
	defer local.Close()
	remote, err := client.Dial("tcp", target)
	if err != nil {
		return
	}
	defer remote.Close()
	pipe(local, remote)
}

// pipe 双向转发
func pipe(a, b io.ReadWriteCloser) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
}
