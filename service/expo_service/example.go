package expo_service

import (
	"context"
	"fmt"
	"log"
	"time"
)

// Example demonstrates how to use the Expo push notification service

// ExampleBasicUsage shows basic usage of the service
func ExampleBasicUsage() {
	// Create a new manager with default configuration
	manager := NewManager()

	// Example push token (replace with real token)
	token := "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]"

	ctx := context.Background()

	// Send a simple notification
	result, err := manager.SendNotification(ctx, token, "Hello", "World!")
	if err != nil {
		log.Printf("Error sending notification: %v", err)
		return
	}

	if result.Success {
		log.Printf("Notification sent successfully, receipt ID: %s", result.ReceiptID)

		// Check receipt after 15 minutes (recommended by Expo)
		go func() {
			time.Sleep(15 * time.Minute)
			receipts, err := manager.CheckReceipts(ctx, []string{result.ReceiptID})
			if err != nil {
				log.Printf("Error checking receipts: %v", err)
				return
			}

			if receipt, exists := receipts[result.ReceiptID]; exists {
				if receipt.Delivered {
					log.Printf("Notification delivered successfully")
				} else {
					log.Printf("Notification delivery failed: %v", receipt.Error)
				}
			}
		}()
	} else {
		log.Printf("Failed to send notification: %v", result.Error)
	}
}

// ExampleBulkNotifications shows how to send notifications to multiple recipients
func ExampleBulkNotifications() {
	manager := NewManager()

	tokens := []string{
		"ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
		"ExponentPushToken[yyyyyyyyyyyyyyyyyyyyyy]",
		"ExponentPushToken[zzzzzzzzzzzzzzzzzzzzzz]",
	}

	ctx := context.Background()

	results, err := manager.SendBulkNotifications(ctx, tokens, "Bulk Message", "This is sent to multiple devices")
	if err != nil {
		log.Printf("Error sending bulk notifications: %v", err)
		return
	}

	// Process results
	var receiptIDs []string
	for _, result := range results {
		if result.Success {
			log.Printf("Sent to %s, receipt ID: %s", result.Token, result.ReceiptID)
			receiptIDs = append(receiptIDs, result.ReceiptID)
		} else {
			log.Printf("Failed to send to %s: %v", result.Token, result.Error)
		}
	}

	// Check receipts later
	if len(receiptIDs) > 0 {
		go func() {
			time.Sleep(15 * time.Minute)
			receipts, err := manager.CheckReceipts(ctx, receiptIDs)
			if err != nil {
				log.Printf("Error checking receipts: %v", err)
				return
			}

			for receiptID, receipt := range receipts {
				if receipt.Delivered {
					log.Printf("Receipt %s: delivered", receiptID)
				} else {
					log.Printf("Receipt %s: failed - %v", receiptID, receipt.Error)
					if receipt.DeviceUnregistered {
						log.Printf("Device unregistered for receipt %s", receiptID)
					}
				}
			}
		}()
	}
}

// ExampleCustomMessage shows how to send a custom message with rich content
func ExampleCustomMessage() {
	manager := NewManager()

	token := "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]"

	// Create a custom message with rich content
	message := &PushMessage{
		To:       []string{token},
		Title:    "Rich Notification",
		Body:     "This notification has an image",
		Sound:    "default",
		Priority: "high",
		Data: map[string]interface{}{
			"customData": "value",
			"actionId":   123,
		},
		RichContent: &RichContent{
			Image: "https://example.com/image.jpg",
		},
		Badge: intPtr(1),
		TTL:   3600, // 1 hour
	}

	ctx := context.Background()

	result, err := manager.SendCustomMessage(ctx, message)
	if err != nil {
		log.Printf("Error sending custom message: %v", err)
		return
	}

	if result.Success {
		log.Printf("Custom message sent successfully, receipt ID: %s", result.ReceiptID)
	} else {
		log.Printf("Failed to send custom message: %v", result.Error)
	}
}

// ExampleWithConfiguration shows how to use custom configuration
func ExampleWithConfiguration() {
	// Create custom configuration
	config := &Config{
		Timeout:         10 * time.Second,
		MaxRetries:      5,
		BaseDelay:       2 * time.Second,
		DefaultSound:    "custom_sound",
		DefaultTTL:      7200, // 2 hours
		DefaultPriority: "high",
		BatchSize:       50,
		MaxConcurrency:  3,
	}

	// Create manager with custom config
	manager := NewManagerWithConfig(config)

	token := "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]"
	ctx := context.Background()

	result, err := manager.SendNotification(ctx, token, "Custom Config", "Using custom configuration")
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if result.Success {
		log.Printf("Sent with custom config, receipt ID: %s", result.ReceiptID)
	} else {
		log.Printf("Failed: %v", result.Error)
	}
}

// ExampleErrorHandling shows error handling and retry mechanisms
func ExampleErrorHandling() {
	manager := NewManager()

	// Invalid token to demonstrate error handling
	invalidToken := "invalid-token"

	ctx := context.Background()

	result, err := manager.SendNotification(ctx, invalidToken, "Test", "This will fail")
	if err != nil {
		log.Printf("Expected error for invalid token: %v", err)
		return
	}

	if !result.Success {
		log.Printf("Send failed as expected: %v", result.Error)
		log.Printf("Retries attempted: %d", result.Retry)
	}
}

// ExampleHealthCheck shows how to perform health checks
func ExampleHealthCheck() {
	manager := NewManager()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.HealthCheck(ctx); err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		log.Printf("Health check passed")
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

// ExampleUsageWithData shows how to send notifications with custom data
func ExampleUsageWithData() {
	manager := NewManager()

	token := "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]"

	// Custom data to include in the notification
	data := map[string]interface{}{
		"userId":     12345,
		"actionType": "message",
		"messageId":  "msg_67890",
		"timestamp":  time.Now().Unix(),
		"metadata": map[string]interface{}{
			"source":  "api",
			"version": "1.0",
		},
	}

	ctx := context.Background()

	result, err := manager.SendNotificationWithData(ctx, token, "New Message", "You have a new message!", data)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if result.Success {
		log.Printf("Notification with data sent, receipt ID: %s", result.ReceiptID)
	} else {
		log.Printf("Failed: %v", result.Error)
	}
}

// ExampleTokenValidation shows token validation
func ExampleTokenValidation() {
	validTokens := []string{
		"ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]",
		"ExpoPushToken[yyyyyyyyyyyyyyyyyyyyyy]",
	}

	invalidTokens := []string{
		"invalid-token",
		"",
		"ExponentPushToken[",
		"too-short",
	}

	fmt.Println("Valid tokens:")
	for _, token := range validTokens {
		fmt.Printf("  %s: %t\n", token, ValidateToken(token))
	}

	fmt.Println("Invalid tokens:")
	for _, token := range invalidTokens {
		fmt.Printf("  %s: %t\n", token, ValidateToken(token))
	}
}
