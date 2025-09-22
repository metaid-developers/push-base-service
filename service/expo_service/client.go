package expo_service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// Expo Push API endpoints
	PushURL    = "https://exp.host/--/api/v2/push/send"
	ReceiptURL = "https://exp.host/--/api/v2/push/getReceipts"

	// Max messages per request
	MaxMessagesPerRequest = 100

	// Default timeout
	DefaultTimeout = 30 * time.Second
)

// Client represents the Expo push notification client
type Client struct {
	httpClient  *http.Client
	timeout     time.Duration
	accessToken string // Expo Access Token
}

// NewClient creates a new Expo push notification client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		timeout: DefaultTimeout,
	}
}

// NewClientWithTimeout creates a new Expo push notification client with custom timeout
func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// NewClientWithAccessToken creates a new Expo push notification client with access token
func NewClientWithAccessToken(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		timeout:     DefaultTimeout,
		accessToken: accessToken,
	}
}

// NewClientWithConfig creates a new Expo push notification client with full config
func NewClientWithConfig(accessToken string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		timeout:     timeout,
		accessToken: accessToken,
	}
}

// PushMessage represents a push notification message
type PushMessage struct {
	To                []string               `json:"to,omitempty"`                // Push tokens
	Title             string                 `json:"title,omitempty"`             // Notification title
	Body              string                 `json:"body"`                        // Notification body
	Data              map[string]interface{} `json:"data,omitempty"`              // Custom data
	Sound             string                 `json:"sound,omitempty"`             // Sound to play
	TTL               int                    `json:"ttl,omitempty"`               // Time to live in seconds
	Expiration        int64                  `json:"expiration,omitempty"`        // Unix timestamp
	Priority          string                 `json:"priority,omitempty"`          // normal or high
	Subtitle          string                 `json:"subtitle,omitempty"`          // iOS subtitle
	Badge             *int                   `json:"badge,omitempty"`             // iOS badge number
	ChannelID         string                 `json:"channelId,omitempty"`         // Android channel ID
	CategoryID        string                 `json:"categoryId,omitempty"`        // Notification category
	MutableContent    bool                   `json:"mutableContent,omitempty"`    // iOS mutable content
	InterruptionLevel string                 `json:"interruptionLevel,omitempty"` // iOS interruption level
	RichContent       *RichContent           `json:"richContent,omitempty"`       // Rich content
}

// RichContent represents rich content for notifications
type RichContent struct {
	Image string `json:"image,omitempty"` // Image URL
}

// PushTicket represents the response from sending a push notification
type PushTicket struct {
	Status  string                 `json:"status"`            // "ok" or "error"
	ID      string                 `json:"id,omitempty"`      // Receipt ID for successful sends
	Message string                 `json:"message,omitempty"` // Error message for failed sends
	Details map[string]interface{} `json:"details,omitempty"` // Additional error details
}

// PushResponse represents the response from the push API
type PushResponse struct {
	Data   []PushTicket `json:"data,omitempty"`
	Errors []APIError   `json:"errors,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PushReceipt represents a push receipt
type PushReceipt struct {
	Status  string          `json:"status"`            // "ok" or "error"
	Message string          `json:"message,omitempty"` // Error message
	Details *ReceiptDetails `json:"details,omitempty"` // Error details
}

// ReceiptDetails contains details about receipt errors
type ReceiptDetails struct {
	Error string `json:"error,omitempty"` // Error type like "DeviceNotRegistered"
}

// ReceiptRequest represents a request to get push receipts
type ReceiptRequest struct {
	IDs []string `json:"ids"`
}

// ReceiptResponse represents the response from the receipt API
type ReceiptResponse struct {
	Data   map[string]PushReceipt `json:"data,omitempty"`
	Errors []APIError             `json:"errors,omitempty"`
}

// SendPushNotification sends a single push notification
func (c *Client) SendPushNotification(ctx context.Context, message *PushMessage) (*PushResponse, error) {
	return c.SendPushNotifications(ctx, []*PushMessage{message})
}

// SendPushNotifications sends multiple push notifications
func (c *Client) SendPushNotifications(ctx context.Context, messages []*PushMessage) (*PushResponse, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to send")
	}

	if len(messages) > MaxMessagesPerRequest {
		return nil, fmt.Errorf("too many messages: %d (max %d)", len(messages), MaxMessagesPerRequest)
	}

	// Convert to JSON
	var payload interface{}
	if len(messages) == 1 {
		payload = messages[0]
	} else {
		payload = messages
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal messages: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", PushURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")

	// 添加 Access Token 认证（如果提供）
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var pushResponse PushResponse
	if err := json.Unmarshal(body, &pushResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &pushResponse, nil
}

// GetPushReceipts retrieves push receipts for the given receipt IDs
func (c *Client) GetPushReceipts(ctx context.Context, receiptIDs []string) (*ReceiptResponse, error) {
	if len(receiptIDs) == 0 {
		return nil, fmt.Errorf("no receipt IDs provided")
	}

	request := ReceiptRequest{IDs: receiptIDs}
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", ReceiptURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 添加 Access Token 认证（如果提供）
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var receiptResponse ReceiptResponse
	if err := json.Unmarshal(body, &receiptResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &receiptResponse, nil
}
