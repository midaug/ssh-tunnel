package tunnel

import (
	"bufio"
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"ssh-tunnel/internal/config"
)

// fakeConnectProxy 起一个只处理 HTTP CONNECT 的假代理，返回其监听地址。
// wantAuth 非空时校验 Proxy-Authorization 头。echoAfter 为 CONNECT 成功后
// 服务端立即写回的数据（用于验证隧道建立后数据可读）。
func fakeConnectProxy(t *testing.T, wantAuth string, statusLine string, echoAfter string) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		br := bufio.NewReader(conn)
		req, err := http.ReadRequest(br)
		if err != nil {
			return
		}
		if req.Method != http.MethodConnect {
			_, _ = conn.Write([]byte("HTTP/1.1 405 Method Not Allowed\r\n\r\n"))
			return
		}
		if wantAuth != "" {
			got := req.Header.Get("Proxy-Authorization")
			if got != wantAuth {
				_, _ = conn.Write([]byte("HTTP/1.1 407 Proxy Authentication Required\r\n\r\n"))
				return
			}
		}
		sl := statusLine
		if sl == "" {
			sl = "HTTP/1.1 200 Connection established"
		}
		_, _ = conn.Write([]byte(sl + "\r\n\r\n"))
		if echoAfter != "" {
			_, _ = conn.Write([]byte(echoAfter))
		}
		// 保持连接直到测试关闭 listener
		_, _ = br.Discard(br.Buffered())
	}()

	return ln.Addr().String()
}

func splitHostPortInt(t *testing.T, addr string) (string, int) {
	t.Helper()
	h, p, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split %s: %v", addr, err)
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		t.Fatalf("parse port %s: %v", p, err)
	}
	return h, port
}

func TestDialHTTPConnect_Success(t *testing.T) {
	addr := fakeConnectProxy(t, "", "", "hello")
	host, port := splitHostPortInt(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := dialViaProxy(ctx, config.Proxy{Type: config.ProxyHTTP, Host: host, Port: port}, "example.com:22")
	if err != nil {
		t.Fatalf("dialViaProxy: %v", err)
	}
	defer conn.Close()

	// CONNECT 成功后应能读到服务端写回的数据
	buf := make([]byte, 5)
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("read after connect: %v", err)
	}
	if string(buf[:n]) != "hello" {
		t.Fatalf("got %q, want hello", string(buf[:n]))
	}
}

func TestDialHTTPConnect_Auth(t *testing.T) {
	cred := "Basic " + base64.StdEncoding.EncodeToString([]byte("u:p"))
	addr := fakeConnectProxy(t, cred, "", "")
	host, port := splitHostPortInt(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := dialViaProxy(ctx, config.Proxy{
		Type: config.ProxyHTTP, Host: host, Port: port, User: "u", Password: "p",
	}, "example.com:22")
	if err != nil {
		t.Fatalf("dialViaProxy with auth: %v", err)
	}
	_ = conn.Close()
}

func TestDialHTTPConnect_AuthFail(t *testing.T) {
	cred := "Basic " + base64.StdEncoding.EncodeToString([]byte("u:right"))
	addr := fakeConnectProxy(t, cred, "", "")
	host, port := splitHostPortInt(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := dialViaProxy(ctx, config.Proxy{
		Type: config.ProxyHTTP, Host: host, Port: port, User: "u", Password: "wrong",
	}, "example.com:22")
	if err == nil {
		t.Fatal("expected error on bad auth, got nil")
	}
	if !strings.Contains(err.Error(), "407") {
		t.Fatalf("expected 407 in error, got: %v", err)
	}
}

func TestDialHTTPConnect_Reject(t *testing.T) {
	addr := fakeConnectProxy(t, "", "HTTP/1.1 502 Bad Gateway", "")
	host, port := splitHostPortInt(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := dialViaProxy(ctx, config.Proxy{Type: config.ProxyHTTP, Host: host, Port: port}, "example.com:22")
	if err == nil {
		t.Fatal("expected error on 502, got nil")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected 502 in error, got: %v", err)
	}
}

func TestDialViaProxy_Disabled(t *testing.T) {
	// 未启用代理时应直连；用一个本地 listener 验证走的是直连路径
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e == nil {
			_ = c.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, err := dialViaProxy(ctx, config.Proxy{}, ln.Addr().String())
	if err != nil {
		t.Fatalf("direct dial: %v", err)
	}
	_ = conn.Close()
}
