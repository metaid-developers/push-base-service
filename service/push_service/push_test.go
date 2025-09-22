package push_service

import (
	"context"
	"fmt"
	"log"
	"push-base-service/service/expo_service"
	"testing"
)

// TestBasicUsage 基本使用测试
func TestBasicUsage(t *testing.T) {
	fmt.Println("=== 基本使用测试 ===")

	// 创建推送管理器
	manager := NewManager()

	// 注册Expo提供者
	err := manager.RegisterExpoProvider(nil) // 使用默认配置
	if err != nil {
		t.Logf("注册Expo提供者失败: %v", err)
		return
	}

	// 启动服务
	if err := manager.Start(); err != nil {
		t.Logf("启动推送服务失败: %v", err)
		return
	}
	defer manager.Stop()

	ctx := context.Background()

	// 1. 设置用户的推送令牌
	metaId := "user123"
	expoToken := "ExponentPushToken[uyx0GKM8MF18TqnRnY3A_j]"

	err = manager.SetUserToken(ctx, metaId, ProviderTypeExpo, expoToken)
	if err != nil {
		t.Logf("设置用户令牌失败: %v", err)
		return
	}

	t.Logf("✅ 已为用户 %s 设置 Expo 推送令牌", metaId)

	// 2. 🎯 核心功能：发送通知给用户的所有平台
	result, err := manager.SendToUser(ctx, metaId, "Hello", "这是一条后端给你的测试消息！")
	if err != nil {
		t.Logf("发送通知失败: %v", err)
		return
	}

	// 3. 处理结果
	t.Logf("📊 发送结果统计:")
	t.Logf("   - 用户数: %d", result.TotalUsers)
	t.Logf("   - 平台数: %d", result.TotalPlatforms)
	t.Logf("   - 成功数: %d", result.SuccessCount)
	t.Logf("   - 失败数: %d", result.FailureCount)
	t.Logf("   - 总耗时: %v", result.Duration)

	for _, pushResult := range result.Results {
		if pushResult.Success {
			t.Logf("✅ 平台 %s: 发送成功，回执ID: %s", pushResult.Platform, pushResult.ReceiptID)
		} else {
			t.Logf("❌ 平台 %s: 发送失败 - %v", pushResult.Platform, pushResult.Error)
		}
	}
}

// TestMultiPlatform 多平台推送测试
func TestMultiPlatform(t *testing.T) {
	fmt.Println("\n=== 多平台推送测试 ===")

	manager := NewManager()

	// 注册多个推送提供者
	manager.RegisterExpoProvider(nil)
	// 这里可以注册其他提供者
	// manager.RegisterFCMProvider(fcmConfig)
	// manager.RegisterAPNSProvider(apnsConfig)

	manager.Start()
	defer manager.Stop()

	ctx := context.Background()
	metaId := "user456"

	// 为用户设置多个平台的推送令牌
	err := manager.SetUserToken(ctx, metaId, ProviderTypeExpo, "ExponentPushToken[xxxxxxxxxxxxxxxxxxxxxx]")
	if err != nil {
		t.Logf("设置Expo令牌失败: %v", err)
		return
	}

	// 如果有其他平台的令牌，也可以设置
	// manager.SetUserToken(ctx, metaId, ProviderTypeFCM, "fcm_token_123")
	// manager.SetUserToken(ctx, metaId, ProviderTypeAPNS, "apns_token_456")

	t.Logf("✅ 已为用户 %s 设置多平台推送令牌", metaId)

	// 查看用户的所有令牌
	userTokens, err := manager.GetUserTokens(ctx, metaId)
	if err != nil {
		t.Logf("获取用户令牌失败: %v", err)
		return
	}

	t.Logf("📱 用户 %s 的推送令牌:", metaId)
	for platform, token := range userTokens.Tokens {
		t.Logf("   - %s: %s", platform, token[:20]+"...")
	}

	// 🎯 一次调用，发送到用户的所有平台
	result, err := manager.SendToUserWithData(ctx, metaId, "多平台通知", "这条消息会发送到你的所有设备！", map[string]interface{}{
		"type":   "multi_platform",
		"userId": metaId,
	})

	if err != nil {
		t.Logf("发送多平台通知失败: %v", err)
		return
	}

	t.Logf("🚀 多平台推送完成!")
	t.Logf("   - 发送到 %d 个平台", result.TotalPlatforms)
	t.Logf("   - 成功 %d 个，失败 %d 个", result.SuccessCount, result.FailureCount)
}

// TestBatchUsers 批量用户推送测试
func TestBatchUsers(t *testing.T) {
	fmt.Println("\n=== 批量用户推送测试 ===")

	manager := NewManager()
	manager.RegisterExpoProvider(nil)
	manager.Start()
	defer manager.Stop()

	ctx := context.Background()

	// 为多个用户设置推送令牌
	users := []struct {
		metaId string
		token  string
	}{
		{"user001", "ExponentPushToken[aaaaaaaaaaaaaaaaaaaaaa]"},
		{"user002", "ExponentPushToken[bbbbbbbbbbbbbbbbbbbbbb]"},
		{"user003", "ExponentPushToken[cccccccccccccccccccccc]"},
	}

	var metaIds []string
	for _, user := range users {
		err := manager.SetUserToken(ctx, user.metaId, ProviderTypeExpo, user.token)
		if err != nil {
			t.Logf("设置用户 %s 令牌失败: %v", user.metaId, err)
			continue
		}
		metaIds = append(metaIds, user.metaId)
	}

	t.Logf("✅ 已为 %d 个用户设置推送令牌", len(metaIds))

	// 🎯 批量发送到所有用户的所有平台
	result, err := manager.SendToUsersWithData(
		ctx,
		metaIds,
		"系统公告",
		"重要系统维护通知，请注意！",
		map[string]interface{}{
			"type":     "system_announcement",
			"priority": "high",
		},
	)

	if err != nil {
		t.Logf("批量发送失败: %v", err)
		return
	}

	t.Logf("📢 批量推送完成!")
	t.Logf("   - 目标用户数: %d", result.TotalUsers)
	t.Logf("   - 涉及平台数: %d", result.TotalPlatforms)
	t.Logf("   - 成功发送: %d", result.SuccessCount)
	t.Logf("   - 发送失败: %d", result.FailureCount)
	t.Logf("   - 总耗时: %v", result.Duration)

	// 按用户统计结果
	userResults := make(map[string][]*PushResult)
	for _, pushResult := range result.Results {
		userResults[pushResult.MetaID] = append(userResults[pushResult.MetaID], pushResult)
	}

	for metaId, results := range userResults {
		successCount := 0
		for _, r := range results {
			if r.Success {
				successCount++
			}
		}
		t.Logf("   👤 用户 %s: %d/%d 平台发送成功", metaId, successCount, len(results))
	}
}

// TestCustomNotification 自定义通知测试
func TestCustomNotification(t *testing.T) {
	fmt.Println("\n=== 自定义通知测试 ===")

	manager := NewManager()

	// 使用自定义Expo配置
	expoConfig := &expo_service.Config{
		Timeout:         15,
		MaxRetries:      5,
		DefaultSound:    "custom_sound",
		DefaultPriority: "high",
	}

	manager.RegisterExpoProvider(expoConfig)
	manager.Start()
	defer manager.Stop()

	ctx := context.Background()
	metaId := "vip_user"

	// 设置VIP用户的推送令牌
	err := manager.SetUserToken(ctx, metaId, ProviderTypeExpo, "ExponentPushToken[vipusertoken123456789]")
	if err != nil {
		t.Logf("设置VIP用户令牌失败: %v", err)
		return
	}

	// 创建自定义通知
	notification := &PushNotification{
		Title:    "🎉 VIP专属通知",
		Body:     "您的VIP特权已激活，享受专属服务！",
		Priority: PriorityHigh,
		Sound:    "vip_notification",
		Badge:    intPtr(1),
		ImageURL: "https://example.com/vip-badge.jpg",
		Data: map[string]interface{}{
			"userLevel":    "vip",
			"specialOffer": true,
			"offerCode":    "VIP2024",
		},
	}

	// 发送自定义通知
	result, err := manager.SendCustomNotificationToUser(ctx, metaId, notification)
	if err != nil {
		t.Logf("发送自定义通知失败: %v", err)
		return
	}

	t.Logf("💎 VIP通知发送完成!")
	for _, pushResult := range result.Results {
		if pushResult.Success {
			t.Logf("✅ 发送成功到 %s 平台", pushResult.Platform)
		}
	}
}

// intPtr 创建int指针的辅助函数
func intPtr(i int) *int {
	return &i
}

// TestAllFeatures 运行所有功能测试
func TestAllFeatures(t *testing.T) {
	log.Println("🚀 推送服务功能测试")
	log.Println("==================")

	t.Run("BasicUsage", TestBasicUsage)
	t.Run("MultiPlatform", TestMultiPlatform)
	t.Run("BatchUsers", TestBatchUsers)
	t.Run("CustomNotification", TestCustomNotification)

	log.Println("\n🎉 所有功能测试运行完成!")
}
