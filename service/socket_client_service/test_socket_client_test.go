package socket_client_service

import (
	"flag"
	"log"
	"testing"
	"time"
)

// TestSocketClientCustom 自定义参数测试示例
func TestSocketClientCustom(t *testing.T) {
	// 这个测试函数展示如何在代码中直接调用
	// 你可以修改这里的参数进行测试

	config := &Config{
		ServerURL:        "https://www.show.now/socket-test",
		ExtraPushAuthKey: "",
		Path:             "/socket.io/",
		Timeout:          10,
	}

	manager := NewManager(config)

	// 设置消息处理器
	manager.SetMessageHandler(func(message *PushMessage) {
		t.Logf("📨 收到推送: %s - %+v", message.Type, message.Data)
	})

	// 设置聊天消息处理器
	manager.SetChatMessageHandler(func(chatMessage *ChatNotificationMessage) {
		t.Logf("💬 收到聊天消息: %s", chatMessage.Type)
		if chatMessage.Data != nil {
			t.Logf("   消息内容: %+v", chatMessage.Data.Message)
			// t.Logf("   转发MetaIDs: %+v", chatMessage.Data.RepostMetaIds)
		}
	})

	// 设置心跳处理器
	manager.SetHeartbeatHandler(func() {
		t.Log("💓 收到服务端心跳")
	})

	manager.SetConnectHandler(func() {
		t.Log("✅ 连接成功")
	})

	manager.SetErrorHandler(func(err error) {
		t.Logf("❌ 连接错误: %v", err)
	})

	// 启动测试
	err := manager.Start()
	if err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	defer manager.Stop()

	// 简短测试
	t.Log("⏳ 测试运行 10 秒...")
	// time.Sleep(10 * time.Second)

	t.Log("🏁 测试完成")
}

// TestSocketClient 测试Socket.IO客户端
func TestSocketClient(t *testing.T) {
	// 定义命令行参数
	var (
		serverURL        = "https://www.show.now"
		extraPushAuthKey = "Hsz9UDmgweqyifkIxS6Q"
		path             = "/socket-test/socket.io/"
		timeout          = 10
		duration         = 60
	)
	flag.Parse()

	log.Printf("🧪 Socket.IO 客户端测试")
	log.Printf("   服务器: %s", serverURL)
	log.Printf("   路径: %s", path)
	log.Printf("   超时: %d秒", timeout)
	log.Printf("   运行时间: %d秒", duration)
	log.Println("=" + repeatStr("=", 50))

	// 创建配置
	config := &Config{
		ServerURL:        serverURL,
		ExtraPushAuthKey: extraPushAuthKey,
		Path:             path,
		Timeout:          timeout,
	}

	// 创建客户端
	manager := NewManager(config)

	// 消息计数器
	messageCount := 0
	chatMessageCount := 0
	heartbeatCount := 0

	// 设置事件处理器
	manager.SetConnectHandler(func() {
		log.Println("✅ 连接成功!")
		log.Println("🎯 开始监听推送消息...")
	})

	manager.SetDisconnectHandler(func() {
		log.Println("❌ 连接断开")
	})

	manager.SetErrorHandler(func(err error) {
		log.Printf("🔥 连接错误: %v", err)
	})

	// 设置聊天消息处理器
	manager.SetChatMessageHandler(func(chatMessage *ChatNotificationMessage) {
		chatMessageCount++
		log.Println("\n" + repeatStr("💬", 20))
		log.Printf("💬 收到第 %d 条聊天消息", chatMessageCount)
		log.Println(repeatStr("💬", 20) + "\n")
	})

	// 设置心跳处理器
	manager.SetHeartbeatHandler(func() {
		heartbeatCount++
		log.Printf("💓 收到第 %d 次服务端心跳", heartbeatCount)
	})

	// 启动客户端
	log.Println("🚀 启动Socket.IO客户端...")
	err := manager.Start()
	if err != nil {
		log.Printf("❌ 启动失败: %v", err)
		return
	}
	defer manager.Stop()

	// 等待连接建立
	log.Println("⏳ 等待连接建立... (3秒)")
	time.Sleep(3 * time.Second)

	// 检查连接状态
	if manager.IsRunning() {
		log.Println("✅ 连接状态: 已连接")

		log.Println("✅ 测试消息已发送")
	} else {
		log.Println("❌ 连接状态: 未连接")
	}

	// 保持运行指定时间
	log.Printf("⏳ 测试运行中，等待推送消息... (%d秒)", duration)

	endTime := time.Now().Add(time.Duration(duration) * time.Second)
	for time.Now().Before(endTime) {
		time.Sleep(5 * time.Second)

		remaining := int(time.Until(endTime).Seconds())
		if manager.IsRunning() {
			log.Printf("   ⏰ 连接正常，剩余 %d 秒 - 已收到 %d 条消息", remaining, messageCount)
		} else {
			log.Printf("   ❌ 连接断开，剩余 %d 秒", remaining)
		}
	}

	// 测试结果
	log.Println("\n📊 测试结果:")
	log.Printf("   运行时间: %d秒", duration)
	log.Printf("   推送消息数: %d", messageCount)
	log.Printf("   聊天消息数: %d", chatMessageCount)
	log.Printf("   心跳次数: %d", heartbeatCount)
	log.Printf("   最终连接状态: %t", manager.IsRunning())

	totalMessages := messageCount + chatMessageCount
	if totalMessages > 0 {
		log.Printf("✅ 测试成功: 共接收到 %d 条消息", totalMessages)
	} else {
		log.Println("⚠️  测试结果: 未收到任何消息")
	}

	if heartbeatCount > 0 {
		log.Printf("💓 心跳正常: 收到 %d 次心跳", heartbeatCount)
	}

	log.Println("🏁 Socket.IO客户端测试完成")
}

// repeatStr 字符串重复函数
func repeatStr(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
