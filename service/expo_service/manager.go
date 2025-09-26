package expo_service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager manages Expo push notification service with configuration
type Manager struct {
	service *Service
	config  *Config
	mu      sync.RWMutex
}

// NewManager creates a new Expo push notification manager with default config
func NewManager() *Manager {
	config := DefaultConfig()
	client := NewClientWithTimeout(config.Timeout)
	service := NewServiceWithConfig(client, config.MaxRetries, config.BaseDelay)

	return &Manager{
		service: service,
		config:  config,
	}
}

// NewManagerWithConfig creates a new manager with custom configuration
func NewManagerWithConfig(config *Config) *Manager {
	if config == nil {
		config = DefaultConfig()
	} else {
		config.ApplyDefaults()
		config.Validate()
	}

	// 根据是否有 Access Token 创建不同的客户端
	var client *Client
	if config.AccessToken != "" {
		client = NewClientWithConfig(config.AccessToken, config.Timeout)
	} else {
		client = NewClientWithTimeout(config.Timeout)
	}

	service := NewServiceWithConfig(client, config.MaxRetries, config.BaseDelay)

	return &Manager{
		service: service,
		config:  config,
	}
}

// SendNotification sends a simple notification
func (m *Manager) SendNotification(ctx context.Context, token, title, body string) (*SendNotificationResult, error) {
	if !ValidateToken(token) {
		return nil, fmt.Errorf("invalid push token: %s", token)
	}

	result := m.service.SendSingleNotification(ctx, token, title, body, nil, "default")
	return result, nil
}

// SendNotificationWithData sends a notification with custom data
func (m *Manager) SendNotificationWithData(ctx context.Context, token, title, body string, data map[string]interface{}) (*SendNotificationResult, error) {
	if !ValidateToken(token) {
		return nil, fmt.Errorf("invalid push token: %s", token)
	}

	result := m.service.SendSingleNotification(ctx, token, title, body, data, "default")
	return result, nil
}

// SendBulkNotifications sends notifications to multiple tokens
func (m *Manager) SendBulkNotifications(ctx context.Context, tokens []string, title, body string) ([]*SendNotificationResult, error) {
	// Validate tokens
	validTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if ValidateToken(token) {
			validTokens = append(validTokens, token)
		}
	}

	if len(validTokens) == 0 {
		return nil, fmt.Errorf("no valid push tokens provided")
	}

	results := m.service.SendBulkNotifications(ctx, validTokens, title, body, nil)
	return results, nil
}

// SendBulkNotificationsWithData sends notifications with custom data to multiple tokens
func (m *Manager) SendBulkNotificationsWithData(ctx context.Context, tokens []string, title, body string, data map[string]interface{}) ([]*SendNotificationResult, error) {
	// Validate tokens
	validTokens := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if ValidateToken(token) {
			validTokens = append(validTokens, token)
		}
	}

	if len(validTokens) == 0 {
		return nil, fmt.Errorf("no valid push tokens provided")
	}

	results := m.service.SendBulkNotifications(ctx, validTokens, title, body, data)
	return results, nil
}

// SendCustomMessage sends a fully customized push message
func (m *Manager) SendCustomMessage(ctx context.Context, message *PushMessage) (*SendNotificationResult, error) {
	if len(message.To) == 0 {
		return nil, fmt.Errorf("no push tokens provided")
	}

	// Validate tokens
	validTokens := make([]string, 0, len(message.To))
	for _, token := range message.To {
		if ValidateToken(token) {
			validTokens = append(validTokens, token)
		}
	}

	if len(validTokens) == 0 {
		return nil, fmt.Errorf("no valid push tokens provided")
	}

	// Apply default values from config
	m.applyDefaults(message)

	// For single token, use single notification method
	if len(validTokens) == 1 {
		message.To = validTokens
		return m.service.SendSingleNotification(ctx, validTokens[0], message.Title, message.Body, message.Data, message.Sound), nil
	}

	// For multiple tokens, we need to send individually or create multiple messages
	// For simplicity, we'll send to the first token only in this method
	// Use SendBulkCustomMessages for multiple tokens
	message.To = []string{validTokens[0]}
	return m.service.SendSingleNotification(ctx, validTokens[0], message.Title, message.Body, message.Data, message.Sound), nil
}

// SendBulkCustomMessages sends custom messages to multiple recipients
func (m *Manager) SendBulkCustomMessages(ctx context.Context, messages []*PushMessage) ([]*SendNotificationResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	var allResults []*SendNotificationResult

	// Process messages in batches
	batchSize := m.config.BatchSize
	for i := 0; i < len(messages); i += batchSize {
		end := i + batchSize
		if end > len(messages) {
			end = len(messages)
		}

		batch := messages[i:end]

		// Apply defaults and validate each message in batch
		validMessages := make([]*PushMessage, 0, len(batch))
		for _, msg := range batch {
			if len(msg.To) > 0 && ValidateToken(msg.To[0]) {
				m.applyDefaults(msg)
				validMessages = append(validMessages, msg)
			}
		}

		if len(validMessages) == 0 {
			continue
		}

		// Send batch
		response, err := m.service.client.SendPushNotifications(ctx, validMessages)
		if err != nil {
			// Create error results for all messages in batch
			for _, msg := range validMessages {
				result := &SendNotificationResult{
					Success: false,
					Error:   err,
					Token:   msg.To[0],
				}
				allResults = append(allResults, result)
			}
			continue
		}

		// Process results
		for i, ticket := range response.Data {
			if i >= len(validMessages) {
				break
			}

			result := &SendNotificationResult{
				Token: validMessages[i].To[0],
			}

			if ticket.Status == "ok" {
				result.Success = true
				result.ReceiptID = ticket.ID
			} else {
				result.Error = fmt.Errorf("push failed: %s", ticket.Message)
			}

			allResults = append(allResults, result)
		}
	}

	return allResults, nil
}

// CheckReceipts checks the delivery status of sent notifications
func (m *Manager) CheckReceipts(ctx context.Context, receiptIDs []string) (map[string]*ReceiptResult, error) {
	return m.service.CheckReceipts(ctx, receiptIDs)
}

// applyDefaults applies configuration defaults to a message
func (m *Manager) applyDefaults(message *PushMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if message.Sound == "" {
		message.Sound = m.config.DefaultSound
	}
	if message.TTL == 0 {
		message.TTL = m.config.DefaultTTL
	}
	if message.Priority == "" {
		message.Priority = m.config.DefaultPriority
	}
}

// UpdateConfig updates the manager configuration
func (m *Manager) UpdateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update config
	m.config = config

	// Recreate client and service with new config
	client := NewClientWithTimeout(config.Timeout)
	m.service = NewServiceWithConfig(client, config.MaxRetries, config.BaseDelay)

	return nil
}

// GetConfig returns a copy of the current configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *m.config
	return &configCopy
}

// HealthCheck performs a basic health check of the service
func (m *Manager) HealthCheck(ctx context.Context) error {
	// Create a test message with invalid token to check API connectivity
	testMessage := &PushMessage{
		To:    []string{"ExponentPushToken[invalid-token-for-health-check]"},
		Title: "Health Check",
		Body:  "This is a health check message",
	}

	// Set a short timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := m.service.client.SendPushNotification(healthCtx, testMessage)

	// We expect this to fail with an API response, not a network error
	// If we get a response (even an error response), the service is healthy
	if err != nil {
		// Check if it's a network error vs API error
		// API errors are acceptable for health check
		return fmt.Errorf("health check failed: %w", err)
	}

	return nil
}
