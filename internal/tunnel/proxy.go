package tunnel

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"

	"ssh-tunnel/internal/config"
	"golang.org/x/net/proxy"
)

// dialViaProxy 通过配置的代理建立到 targetAddr 的 TCP 连接。
// 代理未启用时直接用 net.Dialer 拨号。所有拨号均受 ctx 控制。
func dialViaProxy(ctx context.Context, p config.Proxy, targetAddr string) (net.Conn, error) {
	if !p.Enabled() {
		return (&net.Dialer{}).DialContext(ctx, "tcp", targetAddr)
	}

	proxyPort := p.Port
	if proxyPort == 0 {
		switch p.Type {
		case config.ProxySOCKS5:
			proxyPort = 1080
		case config.ProxyHTTPS:
			proxyPort = 443
		default:
			proxyPort = 8080
		}
	}
	proxyAddr := fmt.Sprintf("%s:%d", p.Host, proxyPort)

	switch p.Type {
	case config.ProxySOCKS5:
		return dialSOCKS5(ctx, proxyAddr, p, targetAddr)
	case config.ProxyHTTP, config.ProxyHTTPS:
		return dialHTTPConnect(ctx, proxyAddr, p, targetAddr)
	default:
		return nil, fmt.Errorf("不支持的代理类型: %s", p.Type)
	}
}

// dialSOCKS5 通过 SOCKS5 代理拨号，支持用户名密码认证。
func dialSOCKS5(ctx context.Context, proxyAddr string, p config.Proxy, targetAddr string) (net.Conn, error) {
	var auth *proxy.Auth
	if p.User != "" {
		auth = &proxy.Auth{User: p.User, Password: p.Password}
	}
	d, err := proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	if err != nil {
		return nil, fmt.Errorf("创建 SOCKS5 代理拨号器失败: %w", err)
	}
	if cd, ok := d.(proxy.ContextDialer); ok {
		return cd.DialContext(ctx, "tcp", targetAddr)
	}
	// 回退：无 context 支持时，用 goroutine + ctx 兜底取消
	type result struct {
		conn net.Conn
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		c, e := d.Dial("tcp", targetAddr)
		ch <- result{c, e}
	}()
	select {
	case <-ctx.Done():
		go func() {
			if r := <-ch; r.conn != nil {
				_ = r.conn.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("SOCKS5 代理连接 %s 失败: %w", targetAddr, r.err)
		}
		return r.conn, nil
	}
}

// dialHTTPConnect 通过 HTTP(S) 代理的 CONNECT 方法建立隧道，支持 Basic 认证。
func dialHTTPConnect(ctx context.Context, proxyAddr string, p config.Proxy, targetAddr string) (net.Conn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("连接代理 %s 失败: %w", proxyAddr, err)
	}

	// HTTPS 代理：与代理之间的连接需要先做 TLS 握手
	if p.Type == config.ProxyHTTPS {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: p.Host})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return nil, fmt.Errorf("代理 TLS 握手失败: %w", err)
		}
		conn = tlsConn
	}

	// ctx 取消时关闭连接，打断下面的读写
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)
	if p.User != "" {
		cred := base64.StdEncoding.EncodeToString([]byte(p.User + ":" + p.Password))
		req += "Proxy-Authorization: Basic " + cred + "\r\n"
	}
	req += "\r\n"

	if _, err := conn.Write([]byte(req)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("发送 CONNECT 请求失败: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("读取代理响应失败: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("代理拒绝 CONNECT: %s", resp.Status)
	}
	// br 可能已缓存了部分数据，但 CONNECT 成功后正文紧跟 SSH 握手，
	// 用 http.ReadResponse 读到 200 后 body 为空，剩余数据仍在底层 conn。
	// 若 br 中残留了预读数据，需要用带缓冲的包装。这里用 bufferedConn 兜底。
	if br.Buffered() > 0 {
		peeked, _ := br.Peek(br.Buffered())
		return &bufferedConn{Conn: conn, buf: peeked}, nil
	}
	return conn, nil
}

// bufferedConn 在 CONNECT 响应读取后，把 bufio 中残留的预读字节接回到读流前面。
type bufferedConn struct {
	net.Conn
	buf []byte
}

func (c *bufferedConn) Read(b []byte) (int, error) {
	if len(c.buf) > 0 {
		n := copy(b, c.buf)
		c.buf = c.buf[n:]
		return n, nil
	}
	return c.Conn.Read(b)
}
