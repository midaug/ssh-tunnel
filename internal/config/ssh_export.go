package config

import (
	"fmt"
	"strings"
)

// ToSSHCommand 把隧道配置转换为等价的 ssh 命令行
// 例: ssh -N -L 8080:localhost:80 -i ~/.ssh/id_rsa user@host -p 22
func ToSSHCommand(t Tunnel) string {
	var parts []string
	parts = append(parts, "ssh", "-N")

	for _, f := range t.Forwards {
		switch f.Type {
		case ForwardLocal:
			parts = append(parts, "-L", forwardLocalArg(f))
		case ForwardRemote:
			parts = append(parts, "-R", forwardRemoteArg(f))
		case ForwardDynamic:
			parts = append(parts, "-D", f.Listen)
		}
	}

	if t.AuthType == AuthKey && t.KeyPath != "" {
		parts = append(parts, "-i", t.KeyPath)
	}

	if t.Port != 0 && t.Port != 22 {
		parts = append(parts, "-p", fmt.Sprintf("%d", t.Port))
	}

	parts = append(parts, fmt.Sprintf("%s@%s", t.User, t.Host))
	return strings.Join(parts, " ")
}

// forwardLocalArg 还原 -L 参数: [bind:]port:host:hport
func forwardLocalArg(f Forward) string {
	listen := f.Listen
	target := f.Target
	lh, lp := splitHostPort(listen)
	th, tp := splitHostPort(target)
	if lh == "" || lh == "127.0.0.1" || lh == "localhost" {
		// 简写：省略 bind
		return fmt.Sprintf("%s:%s:%s", lp, th, tp)
	}
	return fmt.Sprintf("%s:%s:%s:%s", lh, lp, th, tp)
}

// forwardRemoteArg 还原 -R 参数
func forwardRemoteArg(f Forward) string {
	listen := f.Listen
	target := f.Target
	lh, lp := splitHostPort(listen)
	th, tp := splitHostPort(target)
	if lh == "" || lh == "127.0.0.1" || lh == "localhost" {
		return fmt.Sprintf("%s:%s:%s", lp, th, tp)
	}
	return fmt.Sprintf("%s:%s:%s:%s", lh, lp, th, tp)
}

func splitHostPort(addr string) (host, port string) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return "", addr
	}
	return addr[:idx], addr[idx+1:]
}
