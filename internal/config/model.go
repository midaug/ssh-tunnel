package config

// 隧道状态
type Status string

const (
	StatusStopped   Status = "stopped"
	StatusConnecting Status = "connecting"
	StatusRunning   Status = "running"
	StatusError     Status = "error"
)

// 转发类型
type ForwardType string

const (
	ForwardLocal   ForwardType = "L" // -L 本地端口转发
	ForwardRemote  ForwardType = "R" // -R 远程端口转发
	ForwardDynamic ForwardType = "D" // -D 动态 SOCKS5
)

// 认证类型
type AuthType string

const (
	AuthPassword AuthType = "password"
	AuthKey      AuthType = "key"
)

// 代理类型
type ProxyType string

const (
	ProxyNone   ProxyType = ""       // 不使用代理
	ProxyHTTP   ProxyType = "http"   // HTTP CONNECT
	ProxyHTTPS  ProxyType = "https"  // HTTPS CONNECT（到代理的连接走 TLS）
	ProxySOCKS5 ProxyType = "socks5" // SOCKS5
)

// Proxy 代理配置。到 SSH 服务器的 TCP 连接先经过该代理建立。
type Proxy struct {
	Type     ProxyType `json:"type,omitempty"` // 空 = 不走代理
	Host     string    `json:"host,omitempty"`
	Port     int       `json:"port,omitempty"`
	User     string    `json:"user,omitempty"`     // 可选，代理认证用户名
	Password string    `json:"password,omitempty"` // 可选，代理认证密码
}

// Enabled 判断是否启用代理
func (p Proxy) Enabled() bool { return p.Type != ProxyNone && p.Host != "" }

// Forward 单条端口转发规则
// 语义统一：Listen = 监听地址，Target = 转发目标地址
//   Local 转发 (-L):  在本地 Listen，转发到远端 Target
//   Remote 转发 (-R): 在远端 Listen（SSH 服务器侧），转发到本地 Target
//   Dynamic 转发 (-D): 在本地 Listen，按 SOCKS5 协议解析 Target
type Forward struct {
	Type   ForwardType `json:"type"`
	Listen string      `json:"listen"` // 监听地址，如 127.0.0.1:8080 或 8080
	Target string      `json:"target"` // 目标地址，如 localhost:80（动态转发留空）
}

// Tunnel 单条 SSH 隧道配置
type Tunnel struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	Host             string      `json:"host"`
	Port             int         `json:"port"`
	User             string      `json:"user"`
	AuthType         AuthType    `json:"authType"`
	Password         string      `json:"password,omitempty"`
	KeyPath          string      `json:"keyPath,omitempty"`
	KeyPassphrase    string      `json:"keyPassphrase,omitempty"`
	Proxy            Proxy       `json:"proxy,omitempty"` // 代理配置，零值表示不走代理
	Forwards         []Forward   `json:"forwards"`
	AutoReconnect    bool        `json:"autoReconnect"`
	ReconnectMinMS   int         `json:"reconnectMinMs"`   // 初始退避毫秒
	ReconnectMaxMS   int         `json:"reconnectMaxMs"`   // 最大退避毫秒
	ServerAliveInterval int      `json:"serverAliveInterval"` // 心跳秒，0 不发
	// 运行时状态（不持久化到导出文件，但保存到本地 config 便于重启后查看）
	Status    Status    `json:"status,omitempty"`
	LastError string    `json:"lastError,omitempty"`
	StartedAt string    `json:"startedAt,omitempty"`
}

// Settings 全局设置
type Settings struct {
	Autostart      bool   `json:"autostart"`
	Theme          string `json:"theme"` // light | dark | system
	MinimizeToTray bool   `json:"minimizeToTray"`
	StartMinimized bool   `json:"startMinimized"`
	HideFromDock   bool   `json:"hideFromDock"` // 不显示在 Dock 栏（仅菜单栏）
}

// Config 整体配置文件
type Config struct {
	Version  string    `json:"version"`
	Tunnels  []Tunnel  `json:"tunnels"`
	Settings Settings  `json:"settings"`
}

// 默认值
func defaultTunnel() Tunnel {
	return Tunnel{
		Port:               22,
		AuthType:           AuthKey,
		AutoReconnect:      true,
		ReconnectMinMS:     1000,
		ReconnectMaxMS:     30000,
		ServerAliveInterval: 30,
	}
}

// NewDefaultTunnel 返回带默认值的 Tunnel，用于前端新建
func NewDefaultTunnel() Tunnel { return defaultTunnel() }

const CurrentVersion = "1"
