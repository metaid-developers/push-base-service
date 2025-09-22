package push_service

import (
	"context"
	"time"
)

// PushProvider 定义推送提供者接口
type PushProvider interface {
	// GetName 返回提供者名称
	GetName() string

	// SendNotification 发送单个通知
	SendNotification(ctx context.Context, token string, notification *PushNotification) (*PushResult, error)

	// ValidateToken 验证推送令牌格式
	ValidateToken(token string) bool

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error
}

// UserTokenStore 用户令牌存储接口
type UserTokenStore interface {
	// GetUserTokens 根据metaId获取用户的所有推送令牌
	GetUserTokens(ctx context.Context, metaId string) (*UserPushTokens, error)

	// SetUserToken 设置用户在指定平台的推送令牌
	SetUserToken(ctx context.Context, metaId string, platform string, token string) error

	// RemoveUserToken 移除用户在指定平台的推送令牌
	RemoveUserToken(ctx context.Context, metaId string, platform string) error

	// GetAllUserTokens 获取所有用户的令牌（用于批量推送）
	GetAllUserTokens(ctx context.Context, metaIds []string) (map[string]*UserPushTokens, error)
}

// UserPushTokens 用户推送令牌信息
type UserPushTokens struct {
	MetaID    string            `json:"metaId" binding:"required"` // 用户唯一标识
	Tokens    map[string]string `json:"tokens"`                    // 平台->令牌映射 {"expo": "ExponentPushToken[...]", "fcm": "fcm_token_123"}
	UpdatedAt time.Time         `json:"updatedAt"`                 // 最后更新时间
}

// PushNotification 推送通知内容
type PushNotification struct {
	Title    string                 `json:"title" binding:"required"` // 通知标题
	Body     string                 `json:"body" binding:"required"`  // 通知内容
	Data     map[string]interface{} `json:"data,omitempty"`           // 自定义数据
	Sound    string                 `json:"sound,omitempty"`          // 声音
	Badge    *int                   `json:"badge,omitempty"`          // 徽章数字
	ImageURL string                 `json:"imageUrl,omitempty"`       // 图片URL
	Priority string                 `json:"priority,omitempty"`       // 优先级 (normal/high)
}

// PushResult 推送结果
type PushResult struct {
	MetaID    string        `json:"metaId"`              // 用户MetaID
	Platform  string        `json:"platform"`            // 推送平台
	Token     string        `json:"token"`               // 推送令牌
	Success   bool          `json:"success"`             // 是否成功
	ReceiptID string        `json:"receiptId,omitempty"` // 回执ID
	Error     error         `json:"error,omitempty"`     // 错误信息
	Duration  time.Duration `json:"duration"`            // 处理耗时
	Timestamp time.Time     `json:"timestamp"`           // 时间戳
}

// BatchPushResult 批量推送结果
type BatchPushResult struct {
	TotalUsers     int           `json:"totalUsers"`     // 总用户数
	TotalPlatforms int           `json:"totalPlatforms"` // 总平台数
	SuccessCount   int           `json:"successCount"`   // 成功数
	FailureCount   int           `json:"failureCount"`   // 失败数
	Results        []*PushResult `json:"results"`        // 详细结果
	Duration       time.Duration `json:"duration"`       // 总耗时
	Timestamp      time.Time     `json:"timestamp"`      // 时间戳
}

// PushService 推送服务接口
type PushService interface {
	// SendToUser 发送通知给指定用户的所有平台
	SendToUser(ctx context.Context, metaId string, notification *PushNotification) (*BatchPushResult, error)

	// SendToUsers 批量发送通知给多个用户的所有平台
	SendToUsers(ctx context.Context, metaIds []string, notification *PushNotification) (*BatchPushResult, error)

	// RegisterProvider 注册推送提供者
	RegisterProvider(provider PushProvider) error

	// SetUserTokenStore 设置用户令牌存储
	SetUserTokenStore(store UserTokenStore)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) map[string]error

	// Start 启动服务
	Start() error

	// Stop 停止服务
	Stop() error
}

// MemoryTokenStore 内存存储实现（用于测试和简单场景）
type MemoryTokenStore struct {
	tokens map[string]*UserPushTokens
}

// NewMemoryTokenStore 创建内存存储
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens: make(map[string]*UserPushTokens),
	}
}

// GetUserTokens 获取用户令牌
func (m *MemoryTokenStore) GetUserTokens(ctx context.Context, metaId string) (*UserPushTokens, error) {
	if tokens, exists := m.tokens[metaId]; exists {
		return tokens, nil
	}
	return &UserPushTokens{
		MetaID: metaId,
		Tokens: make(map[string]string),
	}, nil
}

// SetUserToken 设置用户令牌
func (m *MemoryTokenStore) SetUserToken(ctx context.Context, metaId string, platform string, token string) error {
	if m.tokens[metaId] == nil {
		m.tokens[metaId] = &UserPushTokens{
			MetaID: metaId,
			Tokens: make(map[string]string),
		}
	}
	m.tokens[metaId].Tokens[platform] = token
	m.tokens[metaId].UpdatedAt = time.Now()
	return nil
}

// RemoveUserToken 移除用户令牌
func (m *MemoryTokenStore) RemoveUserToken(ctx context.Context, metaId string, platform string) error {
	if m.tokens[metaId] != nil {
		delete(m.tokens[metaId].Tokens, platform)
		m.tokens[metaId].UpdatedAt = time.Now()
	}
	return nil
}

// GetAllUserTokens 获取多个用户的令牌
func (m *MemoryTokenStore) GetAllUserTokens(ctx context.Context, metaIds []string) (map[string]*UserPushTokens, error) {
	result := make(map[string]*UserPushTokens)
	for _, metaId := range metaIds {
		if tokens, exists := m.tokens[metaId]; exists {
			result[metaId] = tokens
		} else {
			result[metaId] = &UserPushTokens{
				MetaID: metaId,
				Tokens: make(map[string]string),
			}
		}
	}
	return result, nil
}

// 常量定义
const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"

	ProviderTypeExpo = "expo"
	ProviderTypeFCM  = "fcm"
	ProviderTypeAPNS = "apns"
)
