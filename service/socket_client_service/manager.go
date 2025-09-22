package socket_client_service

import (
	"errors"
	"log"
	"sync"
)

// Manager 简化的Socket.IO客户端管理器
type Manager struct {
	client *Client
	config *Config
	mu     sync.RWMutex
}

// NewManager 创建管理器
func NewManager(config *Config) *Manager {
	return &Manager{
		config: config,
		client: NewClient(config),
	}
}

// Start 启动Socket.IO客户端
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 设置默认回调函数
	m.client.OnConnect = func() {
		log.Printf("🚀 Socket.IO client connected for ExtraPushAuthKey: %s", m.config.ExtraPushAuthKey)
	}

	m.client.OnDisconnect = func() {
		log.Printf("📴 Socket.IO client disconnected")
	}

	m.client.OnError = func(err error) {
		log.Printf("🔥 Socket.IO client error: %v", err)
	}

	// 设置默认消息处理器
	if m.client.OnMessage == nil {
		m.client.OnMessage = func(message *PushMessage) {
			log.Printf("📨 Received push message:")
			log.Printf("   Type: %s", message.Type)
			if message.Data != nil {
				log.Printf("   Data: %+v", message.Data)
			}
		}
	}

	return m.client.Start()
}

// Stop 停止Socket.IO客户端
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		m.client.Stop()
	}
}

// IsRunning 检查是否运行中
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.client != nil && m.client.IsConnected()
}

// SetMessageHandler 设置消息处理器
func (m *Manager) SetMessageHandler(handler func(*PushMessage)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnMessage = handler
}

// SetChatMessageHandler 设置聊天消息处理器
func (m *Manager) SetChatMessageHandler(handler func(*ChatNotificationMessage)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnChatNotificationMessage = handler
}

// SetConnectHandler 设置连接处理器
func (m *Manager) SetConnectHandler(handler func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnConnect = handler
}

// SetDisconnectHandler 设置断开连接处理器
func (m *Manager) SetDisconnectHandler(handler func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnDisconnect = handler
}

// SetErrorHandler 设置错误处理器
func (m *Manager) SetErrorHandler(handler func(error)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnError = handler
}

// SetHeartbeatHandler 设置心跳处理器
func (m *Manager) SetHeartbeatHandler(handler func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.client.OnHeartbeat = handler
}

// SendMessage 发送消息
func (m *Manager) SendMessage(event string, data interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.client == nil {
		log.Printf("❌ Client not initialized")
		return errors.New("client not initialized")
	}

	return m.client.SendMessage(event, data)
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.config
}
