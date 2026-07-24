package config

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/mattn/go-shellwords"
)

// ParseSSHCommand 将 ssh 命令行解析为 Tunnel 配置
// 支持: ssh [-L listen:targetHost:targetPort] [-R listen:targetHost:targetPort]
//            [-D listen] [-p port] [-i keyfile] [-N] [-f] [-o option=value] user@host
func ParseSSHCommand(cmdline string) (Tunnel, error) {
	args, err := shellwords.Parse(cmdline)
	if err != nil {
		return Tunnel{}, fmt.Errorf("命令解析失败: %w", err)
	}
	// 去掉开头的 ssh
	if len(args) > 0 && (args[0] == "ssh" || args[0] == "ssh.exe") {
		args = args[1:]
	}

	t := NewDefaultTunnel()
	t.Forwards = nil
	t.Name = ""

	i := 0
	for i < len(args) {
		a := args[i]
		switch {
		case a == "-L":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-L 缺少参数")
			}
			i++
			f, err := parseForwardArg(ForwardLocal, args[i])
			if err != nil {
				return Tunnel{}, err
			}
			t.Forwards = append(t.Forwards, f)
		case a == "-R":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-R 缺少参数")
			}
			i++
			f, err := parseForwardArg(ForwardRemote, args[i])
			if err != nil {
				return Tunnel{}, err
			}
			t.Forwards = append(t.Forwards, f)
		case a == "-D":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-D 缺少参数")
			}
			i++
			t.Forwards = append(t.Forwards, Forward{
				Type:   ForwardDynamic,
				Listen: args[i],
			})
		case a == "-p":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-p 缺少参数")
			}
			i++
			p, err := strconv.Atoi(args[i])
			if err != nil {
				return Tunnel{}, fmt.Errorf("无效端口: %s", args[i])
			}
			t.Port = p
		case a == "-i":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-i 缺少参数")
			}
			i++
			t.KeyPath = args[i]
			t.AuthType = AuthKey
		case a == "-N", a == "-f", a == "-n", a == "-T":
			// 忽略这些标志
		case a == "-o":
			if i+1 >= len(args) {
				return Tunnel{}, fmt.Errorf("-o 缺少参数")
			}
			i++
			applyOption(&t, args[i])
		case strings.HasPrefix(a, "-o"):
			// 紧贴形式: -oPort=2222
			applyOption(&t, a[2:])
		case strings.HasPrefix(a, "-"):
			// 其他未知选项，忽略
		default:
			// user@host 或 host
			if u, h, ok := splitUserHost(a); ok {
				t.User = u
				t.Host = h
			} else {
				t.Host = a
			}
		}
		i++
	}

	if t.Host == "" {
		return Tunnel{}, fmt.Errorf("未指定主机")
	}
	if t.User == "" {
		return Tunnel{}, fmt.Errorf("未指定用户")
	}
	if t.Port == 0 {
		t.Port = 22
	}
	if len(t.Forwards) == 0 {
		return Tunnel{}, fmt.Errorf("未指定转发规则 (-L/-R/-D)")
	}
	if t.Name == "" {
		t.Name = fmt.Sprintf("%s@%s", t.User, t.Host)
	}
	return t, nil
}

// parseForwardArg 解析 -L / -R 的参数
// 形式：
//   3 段: port:host:hport              → Listen=port, Target=host:hport
//   4 段: bind:port:host:hport         → Listen=bind:port, Target=host:hport
//   2 段: port:port（host 默认 localhost）
//
// 使用括号感知的切分，支持 IPv6 地址（如 [::1]:8080:localhost:80）。
func parseForwardArg(ftype ForwardType, arg string) (Forward, error) {
	parts := splitColonFields(arg)
	var f Forward
	f.Type = ftype
	switch len(parts) {
	case 3:
		f.Listen = parts[0]
		f.Target = net.JoinHostPort(unbracket(parts[1]), parts[2])
	case 4:
		f.Listen = net.JoinHostPort(unbracket(parts[0]), parts[1])
		f.Target = net.JoinHostPort(unbracket(parts[2]), parts[3])
	case 2:
		// 简写: port:port（host 默认 localhost）
		f.Listen = parts[0]
		f.Target = "localhost:" + parts[1]
	default:
		return f, fmt.Errorf("无法解析转发参数: %s", arg)
	}
	return f, nil
}

// splitColonFields 按 ":" 切分，但忽略 [ ] 括号内的冒号（IPv6 地址）。
func splitColonFields(s string) []string {
	var fields []string
	var cur strings.Builder
	inBracket := false
	for _, r := range s {
		switch {
		case r == '[':
			inBracket = true
			cur.WriteRune(r)
		case r == ']':
			inBracket = false
			cur.WriteRune(r)
		case r == ':' && !inBracket:
			fields = append(fields, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	fields = append(fields, cur.String())
	return fields
}

// unbracket 去掉 IPv6 地址两侧的方括号（[::1] → ::1），无括号则原样返回。
func unbracket(s string) string {
	if len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']' {
		return s[1 : len(s)-1]
	}
	return s
}

func splitUserHost(s string) (user, host string, ok bool) {
	idx := strings.Index(s, "@")
	if idx <= 0 || idx == len(s)-1 {
		return "", "", false
	}
	return s[:idx], s[idx+1:], true
}

// applyOption 应用 -o Key=Value 选项。目前识别 Port 与 IdentityFile，
// 其余选项忽略（端口转发工具无需大部分 ssh 选项）。
func applyOption(t *Tunnel, kv string) {
	eq := strings.Index(kv, "=")
	if eq < 0 {
		return
	}
	key, val := strings.TrimSpace(kv[:eq]), strings.TrimSpace(kv[eq+1:])
	switch strings.ToLower(key) {
	case "port":
		if p, err := strconv.Atoi(val); err == nil {
			t.Port = p
		}
	case "identityfile":
		t.KeyPath = val
		t.AuthType = AuthKey
	}
}
