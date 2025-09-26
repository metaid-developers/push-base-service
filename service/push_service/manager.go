package push_service

import (
	"context"
	"fmt"
	"push-base-service/service/expo_service"
	"sync"
)

// Manager 推送服务管理器
type Manager struct {
	service PushService
	mu      sync.RWMutex
}

// NewManager 创建推送服务管理器
func NewManager() *Manager {
	return &Manager{
		service: NewPushService(),
	}
}

// RegisterExpoProvider 注册Expo推送提供者
func (m *Manager) RegisterExpoProvider(config *expo_service.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	provider := NewExpoProvider(config)
	return m.service.RegisterProvider(provider)
}

// SendToUser 发送通知给指定用户的所有平台
func (m *Manager) SendToUser(ctx context.Context, metaId, title, body string) (*BatchPushResult, error) {
	notification := &PushNotification{
		Title: title,
		Body:  body,
	}

	return m.service.SendToUser(ctx, metaId, notification)
}

// SendToUserWithData 发送带数据的通知给指定用户的所有平台
func (m *Manager) SendToUserWithData(ctx context.Context, metaId, title, body string, data map[string]interface{}) (*BatchPushResult, error) {
	notification := &PushNotification{
		Title: title,
		Body:  body,
		Data:  data,
	}

	return m.service.SendToUser(ctx, metaId, notification)
}

// SendToUsers 批量发送通知给多个用户的所有平台
func (m *Manager) SendToUsers(ctx context.Context, metaIds []string, title, body string) (*BatchPushResult, error) {
	notification := &PushNotification{
		Title: title,
		Body:  body,
	}

	return m.service.SendToUsers(ctx, metaIds, notification)
}

// SendToUsersWithData 批量发送带数据的通知给多个用户的所有平台
func (m *Manager) SendToUsersWithData(ctx context.Context, metaIds []string, title, body string, data map[string]interface{}) (*BatchPushResult, error) {
	notification := &PushNotification{
		Title: title,
		Body:  body,
		Data:  data,
		Sound: "default",
	}

	return m.service.SendToUsers(ctx, metaIds, notification)
}

// SendCustomNotificationToUser 发送自定义通知给指定用户
func (m *Manager) SendCustomNotificationToUser(ctx context.Context, metaId string, notification *PushNotification) (*BatchPushResult, error) {
	return m.service.SendToUser(ctx, metaId, notification)
}

// SendCustomNotificationToUsers 发送自定义通知给多个用户
func (m *Manager) SendCustomNotificationToUsers(ctx context.Context, metaIds []string, notification *PushNotification) (*BatchPushResult, error) {
	return m.service.SendToUsers(ctx, metaIds, notification)
}

// SetUserToken 设置用户在指定平台的推送令牌
func (m *Manager) SetUserToken(ctx context.Context, metaId, platform, token string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if defaultService, ok := m.service.(*DefaultPushService); ok {
		return defaultService.GetTokenStore().SetUserToken(ctx, metaId, platform, token)
	}

	return fmt.Errorf("token store not available")
}

// GetUserTokens 获取用户的所有推送令牌
func (m *Manager) GetUserTokens(ctx context.Context, metaId string) (*UserPushTokens, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if defaultService, ok := m.service.(*DefaultPushService); ok {
		return defaultService.GetTokenStore().GetUserTokens(ctx, metaId)
	}

	return nil, fmt.Errorf("token store not available")
}

// RemoveUserToken 移除用户在指定平台的推送令牌
func (m *Manager) RemoveUserToken(ctx context.Context, metaId, platform string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if defaultService, ok := m.service.(*DefaultPushService); ok {
		return defaultService.GetTokenStore().RemoveUserToken(ctx, metaId, platform)
	}

	return fmt.Errorf("token store not available")
}

// SetTokenStore 设置自定义的令牌存储
func (m *Manager) SetTokenStore(store UserTokenStore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.service.SetUserTokenStore(store)
}

// GetProviders 获取所有注册的提供者
func (m *Manager) GetProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if defaultService, ok := m.service.(*DefaultPushService); ok {
		return defaultService.GetProviders()
	}

	return []string{}
}

// HealthCheck 健康检查
func (m *Manager) HealthCheck(ctx context.Context) map[string]error {
	return m.service.HealthCheck(ctx)
}

// Start 启动服务
func (m *Manager) Start() error {
	return m.service.Start()
}

// Stop 停止服务
func (m *Manager) Stop() error {
	return m.service.Stop()
}
