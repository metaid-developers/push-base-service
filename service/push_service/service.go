package push_service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultPushService 默认推送服务实现
type DefaultPushService struct {
	providers  map[string]PushProvider
	tokenStore UserTokenStore
	mu         sync.RWMutex
	running    bool
}

// NewPushService 创建新的推送服务
func NewPushService() *DefaultPushService {
	return &DefaultPushService{
		providers:  make(map[string]PushProvider),
		tokenStore: NewMemoryTokenStore(), // 默认使用内存存储
	}
}

// SendToUser 发送通知给指定用户的所有平台
func (s *DefaultPushService) SendToUser(ctx context.Context, metaId string, notification *PushNotification) (*BatchPushResult, error) {
	startTime := time.Now()

	// 获取用户的推送令牌
	userTokens, err := s.tokenStore.GetUserTokens(ctx, metaId)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens for metaId %s: %w", metaId, err)
	}

	if len(userTokens.Tokens) == 0 {
		return &BatchPushResult{
			TotalUsers:     1,
			TotalPlatforms: 0,
			SuccessCount:   0,
			FailureCount:   0,
			Results:        []*PushResult{},
			Duration:       time.Since(startTime),
			Timestamp:      time.Now(),
		}, nil
	}

	// 并发发送到所有平台
	var results []*PushResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	s.mu.RLock()
	for platform, token := range userTokens.Tokens {
		if provider, exists := s.providers[platform]; exists {
			wg.Add(1)
			go func(p string, t string, prov PushProvider) {
				defer wg.Done()

				result := s.sendSingleNotification(ctx, metaId, p, t, prov, notification)

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}(platform, token, provider)
		}
	}
	s.mu.RUnlock()

	wg.Wait()

	// 统计结果
	successCount := 0
	failureCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	return &BatchPushResult{
		TotalUsers:     1,
		TotalPlatforms: len(results),
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		Results:        results,
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
	}, nil
}

// SendToUsers 批量发送通知给多个用户的所有平台
func (s *DefaultPushService) SendToUsers(ctx context.Context, metaIds []string, notification *PushNotification) (*BatchPushResult, error) {
	startTime := time.Now()

	if len(metaIds) == 0 {
		return &BatchPushResult{
			TotalUsers:     0,
			TotalPlatforms: 0,
			SuccessCount:   0,
			FailureCount:   0,
			Results:        []*PushResult{},
			Duration:       time.Since(startTime),
			Timestamp:      time.Now(),
		}, nil
	}

	// 获取所有用户的推送令牌
	allUserTokens, err := s.tokenStore.GetAllUserTokens(ctx, metaIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}

	// 并发发送到所有用户的所有平台
	var results []*PushResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	s.mu.RLock()
	for metaId, userTokens := range allUserTokens {
		for platform, token := range userTokens.Tokens {
			if provider, exists := s.providers[platform]; exists {
				wg.Add(1)
				go func(mid string, p string, t string, prov PushProvider) {
					defer wg.Done()

					result := s.sendSingleNotification(ctx, mid, p, t, prov, notification)

					mu.Lock()
					results = append(results, result)
					mu.Unlock()
				}(metaId, platform, token, provider)
			}
		}
	}
	s.mu.RUnlock()

	wg.Wait()

	// 统计结果
	successCount := 0
	failureCount := 0
	platformCount := 0

	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// 计算涉及的平台数
	platforms := make(map[string]bool)
	for _, result := range results {
		platforms[result.Platform] = true
	}
	platformCount = len(platforms)

	return &BatchPushResult{
		TotalUsers:     len(metaIds),
		TotalPlatforms: platformCount,
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		Results:        results,
		Duration:       time.Since(startTime),
		Timestamp:      time.Now(),
	}, nil
}

// sendSingleNotification 发送单个通知（内部方法）
func (s *DefaultPushService) sendSingleNotification(ctx context.Context, metaId, platform, token string, provider PushProvider, notification *PushNotification) *PushResult {
	startTime := time.Now()

	result := &PushResult{
		MetaID:    metaId,
		Platform:  platform,
		Token:     token,
		Timestamp: time.Now(),
	}

	// 验证令牌
	if !provider.ValidateToken(token) {
		result.Error = fmt.Errorf("invalid token for platform %s", platform)
		result.Duration = time.Since(startTime)
		return result
	}

	// 发送通知
	providerResult, err := provider.SendNotification(ctx, token, notification)
	if err != nil {
		result.Error = err
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = providerResult.Success
	result.ReceiptID = providerResult.ReceiptID
	result.Error = providerResult.Error
	result.Duration = time.Since(startTime)

	return result
}

// RegisterProvider 注册推送提供者
func (s *DefaultPushService) RegisterProvider(provider PushProvider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	name := provider.GetName()
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	s.providers[name] = provider
	return nil
}

// SetUserTokenStore 设置用户令牌存储
func (s *DefaultPushService) SetUserTokenStore(store UserTokenStore) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if store != nil {
		s.tokenStore = store
	}
}

// HealthCheck 健康检查
func (s *DefaultPushService) HealthCheck(ctx context.Context) map[string]error {
	s.mu.RLock()
	providers := make(map[string]PushProvider)
	for name, provider := range s.providers {
		providers[name] = provider
	}
	s.mu.RUnlock()

	results := make(map[string]error)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 并发检查所有提供者
	for name, provider := range providers {
		wg.Add(1)
		go func(n string, p PushProvider) {
			defer wg.Done()

			err := p.HealthCheck(ctx)
			mu.Lock()
			results[n] = err
			mu.Unlock()
		}(name, provider)
	}

	wg.Wait()

	return results
}

// Start 启动服务
func (s *DefaultPushService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("service is already running")
	}

	s.running = true
	return nil
}

// Stop 停止服务
func (s *DefaultPushService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("service is not running")
	}

	s.running = false
	return nil
}

// GetProviders 获取所有注册的提供者名称
func (s *DefaultPushService) GetProviders() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for name := range s.providers {
		names = append(names, name)
	}

	return names
}

// GetTokenStore 获取令牌存储（用于管理用户令牌）
func (s *DefaultPushService) GetTokenStore() UserTokenStore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tokenStore
}
