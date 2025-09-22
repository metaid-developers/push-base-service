package push_service

import (
	"context"
	"push-base-service/service/expo_service"
	"time"
)

// ExpoProvider Expo推送提供者实现
type ExpoProvider struct {
	manager *expo_service.Manager
}

// NewExpoProvider 创建新的Expo推送提供者
func NewExpoProvider(config *expo_service.Config) *ExpoProvider {
	var manager *expo_service.Manager
	if config != nil {
		manager = expo_service.NewManagerWithConfig(config)
	} else {
		manager = expo_service.NewManager()
	}

	return &ExpoProvider{
		manager: manager,
	}
}

// GetName 返回提供者名称
func (p *ExpoProvider) GetName() string {
	return ProviderTypeExpo
}

// SendNotification 发送单个通知
func (p *ExpoProvider) SendNotification(ctx context.Context, token string, notification *PushNotification) (*PushResult, error) {
	startTime := time.Now()

	// 构建Expo消息
	message := p.buildExpoMessage(token, notification)

	// 发送通知
	expoResult, err := p.manager.SendCustomMessage(ctx, message)
	if err != nil {
		return &PushResult{
			Success:   false,
			Token:     token,
			Error:     err,
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}, nil
	}

	// 处理结果
	result := &PushResult{
		Token:     token,
		Success:   expoResult.Success,
		ReceiptID: expoResult.ReceiptID,
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	if !expoResult.Success && expoResult.Error != nil {
		result.Error = expoResult.Error
	}

	return result, nil
}

// ValidateToken 验证推送令牌格式
func (p *ExpoProvider) ValidateToken(token string) bool {
	return expo_service.ValidateToken(token)
}

// HealthCheck 健康检查
func (p *ExpoProvider) HealthCheck(ctx context.Context) error {
	return p.manager.HealthCheck(ctx)
}

// buildExpoMessage 构建Expo消息
func (p *ExpoProvider) buildExpoMessage(token string, notification *PushNotification) *expo_service.PushMessage {
	message := &expo_service.PushMessage{
		To:       []string{token},
		Title:    notification.Title,
		Body:     notification.Body,
		Data:     notification.Data,
		Sound:    notification.Sound,
		Priority: notification.Priority,
	}

	// 设置徽章
	if notification.Badge != nil {
		message.Badge = notification.Badge
	}

	// 设置富内容
	if notification.ImageURL != "" {
		message.RichContent = &expo_service.RichContent{
			Image: notification.ImageURL,
		}
	}

	return message
}
