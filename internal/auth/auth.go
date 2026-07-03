package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// BuildAuthMethods 根据 authType 返回可用的认证方法列表
// 优先尝试提供的密码/密钥；同时静默附加 ssh-agent（若可用）作为后备
func BuildAuthMethods(authType string, password, keyPath, keyPassphrase string) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	switch authType {
	case "password":
		if password == "" {
			return nil, errors.New("密码为空")
		}
		methods = append(methods, ssh.Password(password))
	case "key":
		if keyPath == "" {
			return nil, errors.New("密钥路径为空")
		}
		am, err := publicKeyAuth(keyPath, keyPassphrase)
		if err != nil {
			return nil, err
		}
		methods = append(methods, am)
	default:
		return nil, fmt.Errorf("未知认证类型: %s", authType)
	}

	// 附加 ssh-agent（如有 agent 可用），作为后备认证
	if agent, err := SSHAgent(); err == nil && agent != nil {
		methods = append(methods, agent)
	}
	return methods, nil
}

func publicKeyAuth(keyPath, passphrase string) (ssh.AuthMethod, error) {
	keyPath = expandPath(keyPath)
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("读取密钥失败: %w", err)
	}
	var signer ssh.Signer
	if passphrase == "" {
		signer, err = ssh.ParsePrivateKey(data)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	}
	if err != nil {
		// 给出更友好的提示：可能是需要密码短语
		if strings.Contains(err.Error(), "password protected") || strings.Contains(err.Error(), "encrypted") {
			return nil, errors.New("密钥需要密码短语，请在配置中填写")
		}
		return nil, fmt.Errorf("解析密钥失败: %w", err)
	}
	return ssh.PublicKeys(signer), nil
}
