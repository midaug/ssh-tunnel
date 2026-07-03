package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"

	"ssh-tunnel/internal/config"
	"golang.org/x/crypto/ssh"
)

// startDynamic 动态转发 -D listen
// 在本地监听 listen，实现 SOCKS5 协议，按目标地址通过 ssh 拨号
func (t *Tunnel) startDynamic(client *ssh.Client, f config.Forward) error {
	local := normalizeAddr(f.Listen)
	if local == "" {
		return fmt.Errorf("本地监听地址为空")
	}
	ln, err := net.Listen("tcp", local)
	if err != nil {
		return fmt.Errorf("本地监听 %s 失败: %w", local, err)
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
			go t.handleDynamicConn(client, conn)
		}
	}()
	return nil
}

// handleDynamicConn 处理 SOCKS5 连接
func (t *Tunnel) handleDynamicConn(client *ssh.Client, conn net.Conn) {
	defer conn.Close()
	// SOCKS5 握手：版本 + 方法数 + 方法列表
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return
	}
	if hdr[0] != 5 {
		return
	}
	nmethods := int(hdr[1])
	if _, err := io.ReadFull(conn, make([]byte, nmethods)); err != nil {
		return
	}
	// 回复：不需要认证
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}
	// 读取请求
	req := make([]byte, 4)
	if _, err := io.ReadFull(conn, req); err != nil {
		return
	}
	if req[0] != 5 || req[1] != 1 { // 只支持 CONNECT
		conn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	var host string
	switch req[3] {
	case 1: // IPv4
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	case 3: // 域名
		l := make([]byte, 1)
		if _, err := io.ReadFull(conn, l); err != nil {
			return
		}
		buf := make([]byte, int(l[0]))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		host = string(buf)
	case 4: // IPv6
		buf := make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return
		}
		host = net.IP(buf).String()
	default:
		return
	}
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(portBuf)
	target := net.JoinHostPort(host, strconv.Itoa(int(port)))

	remote, err := client.Dial("tcp", target)
	if err != nil {
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()
	// 成功回复
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	pipe(conn, remote)
}
