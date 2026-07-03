package auth

import (
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHAgent 尝试连接本机 ssh-agent，返回 AuthMethod（无 agent 时返回 nil, err）
func SSHAgent() (ssh.AuthMethod, error) {
	socket := os.Getenv("SSH_AUTH_SOCK")
	if socket == "" {
		return nil, errNoAgent
	}
	conn, err := net.Dial("unix", socket)
	if err != nil {
		return nil, err
	}
	a := agent.NewClient(conn)
	return ssh.PublicKeysCallback(a.Signers), nil
}

var errNoAgent = &agentError{"no ssh-agent available"}

type agentError struct{ msg string }

func (e *agentError) Error() string { return e.msg }

// expandPath 展开 ~ 为家目录
func expandPath(p string) string {
	if p == "" {
		return p
	}
	if p[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			if len(p) == 1 {
				return home
			}
			if p[1] == '/' || os.IsPathSeparator(p[1]) {
				return home + p[1:]
			}
		}
	}
	return p
}
