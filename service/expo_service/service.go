package expo_service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// Service provides high-level push notification functionality with retry logic
type Service struct {
	client     *Client
	maxRetries int
	baseDelay  time.Duration
}

// NewService creates a new Expo push notification service
func NewService() *Service {
	return &Service{
		client:     NewClient(),
		maxRetries: 3,
		baseDelay:  time.Second,
	}
}

// NewServiceWithConfig creates a new service with custom configuration
func NewServiceWithConfig(client *Client, maxRetries int, baseDelay time.Duration) *Service {
	return &Service{
		client:     client,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
	}
}

// SendNotificationResult represents the result of sending a notification
type SendNotificationResult struct {
	Success   bool
	ReceiptID string
	Error     error
	Token     string
	Retry     int
}

// SendSingleNotification sends a notification to a single token with retry logic
func (s *Service) SendSingleNotification(ctx context.Context, token, title, body string, data map[string]interface{}) *SendNotificationResult {
	message := &PushMessage{
		To:    []string{token},
		Title: title,
		Body:  body,
		Data:  data,
	}

	result := &SendNotificationResult{
		Token: token,
	}

	for retry := 0; retry <= s.maxRetries; retry++ {
		result.Retry = retry

		response, err := s.client.SendPushNotification(ctx, message)
		if err != nil {
			if s.shouldRetry(err, retry) {
				s.waitBeforeRetry(retry)
				continue
			}
			result.Error = err
			return result
		}

		if len(response.Errors) > 0 {
			result.Error = fmt.Errorf("API errors: %v", response.Errors)
			return result
		}

		if len(response.Data) > 0 {
			ticket := response.Data[0]
			if ticket.Status == "ok" {
				result.Success = true
				result.ReceiptID = ticket.ID
				return result
			} else {
				result.Error = fmt.Errorf("push failed: %s - %s", ticket.Message, ticket.Details)
				return result
			}
		}

		result.Error = fmt.Errorf("no response data")
		return result
	}

	result.Error = fmt.Errorf("max retries exceeded")
	return result
}

// SendBulkNotifications sends notifications to multiple tokens
func (s *Service) SendBulkNotifications(ctx context.Context, tokens []string, title, body string, data map[string]interface{}) []*SendNotificationResult {
	results := make([]*SendNotificationResult, 0, len(tokens))

	// Split tokens into batches of MaxMessagesPerRequest
	for i := 0; i < len(tokens); i += MaxMessagesPerRequest {
		end := i + MaxMessagesPerRequest
		if end > len(tokens) {
			end = len(tokens)
		}

		batchTokens := tokens[i:end]
		batchResults := s.sendBatch(ctx, batchTokens, title, body, data)
		results = append(results, batchResults...)
	}

	return results
}

// sendBatch sends a batch of notifications
func (s *Service) sendBatch(ctx context.Context, tokens []string, title, body string, data map[string]interface{}) []*SendNotificationResult {
	messages := make([]*PushMessage, len(tokens))
	for i, token := range tokens {
		messages[i] = &PushMessage{
			To:    []string{token},
			Title: title,
			Body:  body,
			Data:  data,
		}
	}

	results := make([]*SendNotificationResult, len(tokens))
	for i, token := range tokens {
		results[i] = &SendNotificationResult{Token: token}
	}

	for retry := 0; retry <= s.maxRetries; retry++ {
		response, err := s.client.SendPushNotifications(ctx, messages)
		if err != nil {
			if s.shouldRetry(err, retry) {
				s.waitBeforeRetry(retry)
				continue
			}
			// Set error for all tokens
			for i := range results {
				if !results[i].Success {
					results[i].Error = err
					results[i].Retry = retry
				}
			}
			return results
		}

		if len(response.Errors) > 0 {
			// Set error for all tokens
			for i := range results {
				if !results[i].Success {
					results[i].Error = fmt.Errorf("API errors: %v", response.Errors)
					results[i].Retry = retry
				}
			}
			return results
		}

		// Process individual results
		for i, ticket := range response.Data {
			if i >= len(results) {
				break
			}

			if ticket.Status == "ok" {
				results[i].Success = true
				results[i].ReceiptID = ticket.ID
				results[i].Retry = retry
			} else {
				results[i].Error = fmt.Errorf("push failed: %s - %s", ticket.Message, ticket.Details)
				results[i].Retry = retry
			}
		}

		return results
	}

	// Max retries exceeded
	for i := range results {
		if !results[i].Success && results[i].Error == nil {
			results[i].Error = fmt.Errorf("max retries exceeded")
			results[i].Retry = s.maxRetries
		}
	}

	return results
}

// CheckReceipts checks the delivery status of sent notifications
func (s *Service) CheckReceipts(ctx context.Context, receiptIDs []string) (map[string]*ReceiptResult, error) {
	if len(receiptIDs) == 0 {
		return make(map[string]*ReceiptResult), nil
	}

	var allResults = make(map[string]*ReceiptResult)

	// Process receipts in batches to avoid API limits
	batchSize := 1000 // Expo doesn't specify a limit, but let's be conservative
	for i := 0; i < len(receiptIDs); i += batchSize {
		end := i + batchSize
		if end > len(receiptIDs) {
			end = len(receiptIDs)
		}

		batch := receiptIDs[i:end]
		batchResults, err := s.checkReceiptsBatch(ctx, batch)
		if err != nil {
			return nil, err
		}

		for id, result := range batchResults {
			allResults[id] = result
		}
	}

	return allResults, nil
}

// ReceiptResult represents the result of checking a receipt
type ReceiptResult struct {
	Status             string
	Delivered          bool
	Error              error
	DeviceUnregistered bool
}

// checkReceiptsBatch checks a batch of receipts
func (s *Service) checkReceiptsBatch(ctx context.Context, receiptIDs []string) (map[string]*ReceiptResult, error) {
	response, err := s.client.GetPushReceipts(ctx, receiptIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API errors: %v", response.Errors)
	}

	results := make(map[string]*ReceiptResult)
	for receiptID, receipt := range response.Data {
		result := &ReceiptResult{
			Status: receipt.Status,
		}

		if receipt.Status == "ok" {
			result.Delivered = true
		} else {
			result.Error = fmt.Errorf("delivery failed: %s", receipt.Message)

			// Check if device is unregistered
			if receipt.Details != nil && receipt.Details.Error == "DeviceNotRegistered" {
				result.DeviceUnregistered = true
			}
		}

		results[receiptID] = result
	}

	return results, nil
}

// shouldRetry determines if an error should trigger a retry
func (s *Service) shouldRetry(err error, retryCount int) bool {
	if retryCount >= s.maxRetries {
		return false
	}

	// Add logic to determine if error is retryable
	// For now, we'll retry on all errors except the last attempt
	return true
}

// waitBeforeRetry implements exponential backoff
func (s *Service) waitBeforeRetry(retryCount int) {
	if retryCount == 0 {
		return
	}

	// Exponential backoff: baseDelay * 2^(retryCount-1)
	delay := time.Duration(float64(s.baseDelay) * math.Pow(2, float64(retryCount-1)))

	// Add some jitter to avoid thundering herd
	jitter := time.Duration(float64(delay) * 0.1)
	delay += jitter

	log.Printf("Waiting %v before retry %d", delay, retryCount)
	time.Sleep(delay)
}

// ValidateToken validates if a token looks like a valid Expo push token
func ValidateToken(token string) bool {
	if len(token) < 10 {
		return false
	}

	// Expo push tokens start with "ExponentPushToken["
	return len(token) > 20 && (token[:18] == "ExponentPushToken[" || token[:14] == "ExpoPushToken[")
}

// CreateSimpleMessage creates a simple push message
func CreateSimpleMessage(token, title, body string) *PushMessage {
	return &PushMessage{
		To:    []string{token},
		Title: title,
		Body:  body,
		Sound: "default",
	}
}

// CreateRichMessage creates a push message with rich content
func CreateRichMessage(token, title, body string, data map[string]interface{}, imageURL string) *PushMessage {
	message := &PushMessage{
		To:    []string{token},
		Title: title,
		Body:  body,
		Data:  data,
		Sound: "default",
	}

	if imageURL != "" {
		message.RichContent = &RichContent{
			Image: imageURL,
		}
	}

	return message
}
