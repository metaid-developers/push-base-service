package pushcenter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"push-base-service/service/pebble_service"
	"push-base-service/service/push_service"
	"push-base-service/service/socket_client_service"
	"slices"
	"sync"
	"time"
)

// PushCenter 推送中心管理器
type PushCenter struct {
	socketManager *socket_client_service.Manager
	pushManager   *push_service.Manager
	config        *Config
	running       bool
	mu            sync.RWMutex
}

// Config 推送中心配置
type Config struct {
	SocketConfig *socket_client_service.Config `yaml:"socket" json:"socket"`
	PebbleConfig *pebble_service.Config        `yaml:"pebble" json:"pebble"`               // Pebble 数据库配置
	EnabledTypes []string                      `yaml:"enabled_types" json:"enabled_types"` // 启用的消息类型
}

// ParsedMessageInfo 解析后的消息信息
type ParsedMessageInfo struct {
	PinId        string `json:"pinId"`        // PIN ID
	GroupId      string `json:"groupId"`      // 群聊ID（群聊消息时使用）
	MetaId       string `json:"metaId"`       // 私聊的MetaId（私聊消息时使用）
	ChatType     string `json:"chatType"`     // 聊天类型：private_chat 或 group_chat
	UserName     string `json:"userName"`     // 用户名
	ChatInfoType int64  `json:"chatInfoType"` // 聊天信息类型：1/23-红包
}

// NewPushCenter 创建推送中心实例
func NewPushCenter(config *Config) *PushCenter {
	// 默认启用所有消息类型
	if len(config.EnabledTypes) == 0 {
		config.EnabledTypes = []string{"private_chat", "group_chat"}
	}

	return &PushCenter{
		socketManager: socket_client_service.NewManager(config.SocketConfig),
		pushManager:   push_service.NewManager(),
		config:        config,
		running:       false,
	}
}

// Initialize 初始化推送中心
func (pc *PushCenter) Initialize() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	log.Printf("🚀 正在初始化推送中心...")

	// 初始化 Pebble 数据库服务
	if pc.config.PebbleConfig != nil {
		if err := pebble_service.InitializeGlobalService(pc.config.PebbleConfig); err != nil {
			log.Printf("❌ 初始化 Pebble 服务失败: %v", err)
			return fmt.Errorf("初始化 Pebble 服务失败: %w", err)
		}
		log.Printf("✅ Pebble 数据库服务已初始化")
	} else {
		// 使用默认配置初始化
		if err := pebble_service.InitializeGlobalService(nil); err != nil {
			log.Printf("❌ 初始化默认 Pebble 服务失败: %v", err)
			return fmt.Errorf("初始化默认 Pebble 服务失败: %w", err)
		}
		log.Printf("✅ 默认 Pebble 数据库服务已初始化")
	}

	// 设置推送服务使用 Pebble 令牌存储
	pebbleTokenStore := pebble_service.NewGlobalPebbleTokenStore()
	if pebbleTokenStore == nil {
		return fmt.Errorf("无法创建 Pebble 令牌存储，全局服务未正确初始化")
	}
	pc.pushManager.SetTokenStore(pebbleTokenStore)
	log.Printf("✅ 推送服务已配置使用 Pebble 令牌存储")

	// 设置 socket 连接处理器
	pc.socketManager.SetConnectHandler(func() {
		log.Printf("✅ Socket 客户端已连接")
	})

	pc.socketManager.SetDisconnectHandler(func() {
		log.Printf("❌ Socket 客户端已断开连接")
	})

	pc.socketManager.SetErrorHandler(func(err error) {
		log.Printf("🔥 Socket 客户端错误: %v", err)
	})

	// 设置聊天消息处理器
	pc.SetChatMessageHandler()

	log.Printf("✅ 推送中心初始化完成")
	return nil
}

// Run 运行推送中心
func (pc *PushCenter) Run() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.running {
		return fmt.Errorf("推送中心已经在运行中")
	}

	log.Printf("🚀 启动推送中心...")

	// 启动 socket 客户端连接
	if err := pc.socketManager.Start(); err != nil {
		log.Printf("❌ 启动 Socket 客户端失败: %v", err)
		return fmt.Errorf("启动 Socket 客户端失败: %w", err)
	}

	// 启动推送服务
	if err := pc.pushManager.Start(); err != nil {
		log.Printf("❌ 启动推送服务失败: %v", err)
		return fmt.Errorf("启动推送服务失败: %w", err)
	}

	pc.running = true
	log.Printf("✅ 推送中心已启动，正在监听消息...")

	return nil
}

// Stop 停止推送中心
func (pc *PushCenter) Stop() error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.running {
		return nil
	}

	log.Printf("🛑 正在停止推送中心...")

	// 停止 socket 客户端
	pc.socketManager.Stop()

	// 停止推送服务
	if err := pc.pushManager.Stop(); err != nil {
		log.Printf("⚠️ 停止推送服务时出现错误: %v", err)
	}

	// 关闭 Pebble 服务
	if err := pebble_service.CloseGlobalService(); err != nil {
		log.Printf("⚠️ 关闭 Pebble 服务时出现错误: %v", err)
	} else {
		log.Printf("✅ Pebble 数据库服务已关闭")
	}

	pc.running = false
	log.Printf("✅ 推送中心已停止")

	return nil
}

// IsRunning 检查推送中心是否正在运行
func (pc *PushCenter) IsRunning() bool {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.running && pc.socketManager.IsRunning()
}

// GetPushManager 获取推送服务管理器
func (pc *PushCenter) GetPushManager() *push_service.Manager {
	return pc.pushManager
}

// SetChatMessageHandler 设置聊天消息处理器
func (pc *PushCenter) SetChatMessageHandler() {
	pc.socketManager.SetChatMessageHandler(func(chatMsg *socket_client_service.ChatNotificationMessage) {
		if chatMsg == nil || chatMsg.Data == nil {
			log.Printf("⚠️ 收到空的聊天消息")
			return
		}

		log.Printf("📨 收到聊天消息: Type=%s", chatMsg.Type)

		// 检查消息类型是否启用
		if !pc.isMessageTypeEnabled(chatMsg.Type) {
			log.Printf("⚠️ 消息类型 %s 未启用，跳过处理", chatMsg.Type)
			return
		}

		// 处理聊天消息并转发推送
		go pc.processChatMessage(chatMsg)
	})
}

// isMessageTypeEnabled 检查消息类型是否启用
func (pc *PushCenter) isMessageTypeEnabled(msgType string) bool {
	for _, enabledType := range pc.config.EnabledTypes {
		if enabledType == msgType {
			return true
		}
	}
	return false
}

// processChatMessage 处理聊天消息
func (pc *PushCenter) processChatMessage(chatMsg *socket_client_service.ChatNotificationMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 解析消息信息，获取 pinId、groupId 和私聊的 metaId
	parsedInfo, err := pc.parseMessageInfo(chatMsg)
	if err != nil {
		log.Printf("❌ 解析消息信息失败: %v", err)
		return
	}

	if parsedInfo.PinId != "" {
		isNotified, err := pebble_service.IsNotifiedPin(parsedInfo.PinId)
		if err != nil {
			log.Printf("❌ 检查PIN通知状态失败: %v", err)
			return
		}
		if isNotified {
			log.Printf("📌 PIN已通知，跳过推送")
			return
		}
	}

	// 提取需要推送的用户ID列表
	metaIds := chatMsg.Data.RepostMetaIds
	if len(metaIds) == 0 {
		log.Printf("⚠️ 没有需要推送的用户ID")
		return
	}

	// 过滤掉已屏蔽该聊天的用户
	filteredMetaIds := pc.filterBlockedUsers(metaIds, parsedInfo)
	// if len(filteredMetaIds) == 0 {
	// 	log.Printf("⚠️ 所有用户都已屏蔽该聊天，跳过推送")
	// 	return
	// }

	// 处理 MentionMetaIds：分类用户
	var mentionMetaIds []string
	if len(chatMsg.Data.MentionMetaIds) > 0 {
		// 过滤被提及的用户（移除已屏蔽的）
		mentionMetaIds = chatMsg.Data.MentionMetaIds
		fmt.Printf("mentionMetaIds: %+v\n", mentionMetaIds)

	}

	// 将用户分为两组：被提及的用户和普通用户
	var mentionedUsers []string
	var normalUsers []string
	mentionedUsers = mentionMetaIds

	//filteredMetaIds里面去重mentionMetaIds,如果有重复的，则只保留一个
	for _, metaId := range filteredMetaIds {
		if !slices.Contains(mentionMetaIds, metaId) {
			normalUsers = append(normalUsers, metaId)
		}
	}

	// 为被提及的用户生成通知（参考 Telegram 的提及消息格式）
	if len(mentionedUsers) > 0 {
		mentionTitle := pc.generateNotificationTitle(chatMsg.Type, true)
		mentionBody := pc.GenerateNotificationBody(chatMsg.Type, parsedInfo.UserName, parsedInfo.ChatInfoType, true, parsedInfo.GroupId)

		// 构造提及消息的自定义数据
		mentionData := map[string]interface{}{
			"type":      chatMsg.Type,
			"message":   chatMsg.Data.Message,
			"timestamp": time.Now().Unix(),
			"pinId":     parsedInfo.PinId,
			"isMention": true,
		}

		// 根据聊天类型添加特定信息
		if parsedInfo.ChatType == "private_chat" && parsedInfo.MetaId != "" {
			mentionData["metaId"] = parsedInfo.MetaId
		} else if parsedInfo.ChatType == "group_chat" && parsedInfo.GroupId != "" {
			mentionData["groupId"] = parsedInfo.GroupId
		}

		log.Printf("🔔 开始推送提及消息给 %d 个用户", len(mentionedUsers))
		mentionResult, err := pc.pushManager.SendToUsersWithData(ctx, mentionedUsers, mentionTitle, mentionBody, mentionData)
		if err != nil {
			log.Printf("❌ 推送提及消息失败: %v", err)
		} else {
			log.Printf("✅ 提及消息推送完成: 总用户=%d, 成功=%d, 失败=%d, 耗时=%v",
				mentionResult.TotalUsers, mentionResult.SuccessCount, mentionResult.FailureCount, mentionResult.Duration)
		}
	}

	// 为普通用户生成通知
	if len(normalUsers) > 0 {
		title := pc.generateNotificationTitle(chatMsg.Type, false)
		body := pc.GenerateNotificationBody(chatMsg.Type, parsedInfo.UserName, parsedInfo.ChatInfoType, false, "")

		// 构造自定义数据，包含解析后的信息
		normalData := map[string]interface{}{
			"type":      chatMsg.Type,
			"message":   chatMsg.Data.Message,
			"timestamp": time.Now().Unix(),
			"pinId":     parsedInfo.PinId,
		}

		// 根据聊天类型添加特定信息
		if parsedInfo.ChatType == "private_chat" && parsedInfo.MetaId != "" {
			normalData["metaId"] = parsedInfo.MetaId
			log.Printf("📱 私聊消息 - 发送者/接收者MetaId: %s, 用户名: %s", parsedInfo.MetaId, parsedInfo.UserName)
		} else if parsedInfo.ChatType == "group_chat" && parsedInfo.GroupId != "" {
			normalData["groupId"] = parsedInfo.GroupId
			log.Printf("👥 群聊消息 - 群组ID: %s, 用户名: %s", parsedInfo.GroupId, parsedInfo.UserName)
		}

		log.Printf("🚀 开始推送普通消息给 %d 个用户", len(normalUsers))
		log.Printf("📋 消息详情 - PinId: %s, ChatType: %s, UserName: %s", parsedInfo.PinId, parsedInfo.ChatType, parsedInfo.UserName)

		// 调用 push_service.SendToUsers 发送推送
		normalResult, err := pc.pushManager.SendToUsersWithData(ctx, normalUsers, title, body, normalData)
		if err != nil {
			log.Printf("❌ 推送普通消息失败: %v", err)
		} else {
			// 记录推送结果
			log.Printf("✅ 普通消息推送完成: 总用户=%d, 成功=%d, 失败=%d, 耗时=%v",
				normalResult.TotalUsers, normalResult.SuccessCount, normalResult.FailureCount, normalResult.Duration)

			// 如果有失败的推送，记录详细信息
			if normalResult.FailureCount > 0 {
				for _, pushResult := range normalResult.Results {
					if !pushResult.Success && pushResult.Error != nil {
						log.Printf("⚠️ 推送失败 - 用户: %s, 平台: %s, 错误: %v",
							pushResult.MetaID, pushResult.Platform, pushResult.Error)
					}
				}
			}
		}
	}

	// 添加已通知PIN记录（使用解析后的 PinId）
	if parsedInfo.PinId != "" {
		go pebble_service.AddNotifiedPin(parsedInfo.PinId)
		log.Printf("📌 已记录PIN通知状态: %s", parsedInfo.PinId)
	} else {
		log.Printf("⚠️ PinId为空，跳过PIN通知记录")
	}
}

// generateNotificationTitle 生成通知标题
func (pc *PushCenter) generateNotificationTitle(msgType string, isMention bool) string {
	if isMention {
		// 提及消息的标题（参考 Telegram）
		switch msgType {
		case "private_chat":
			return "New Mention"
		case "group_chat":
			return "You were mentioned"
		default:
			return "New Mention"
		}
	}

	// 普通消息的标题
	switch msgType {
	case "private_chat":
		return "New Message"
	case "group_chat":
		return "New Message in Group"
	default:
		return "New Message"
	}
}

// GenerateNotificationBody 生成通知内容
func (pc *PushCenter) GenerateNotificationBody(msgType, userName string, chatInfoType int64, isMention bool, groupId string) string {
	if isMention {
		// 提及消息的内容（参考 Telegram 的提及消息格式）
		truncatedName := pc.truncateUserName(userName)
		if truncatedName == "" {
			truncatedName = "Someone"
		}

		switch msgType {
		case "private_chat":
			// 私聊提及："{用户名} mentioned you"
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s mentioned you with a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s mentioned you", truncatedName)
		case "group_chat":
			// 群聊提及："{用户名} mentioned you in {群组名}" 或 "{用户名} mentioned you"
			// 注意：这里 groupId 是群组ID，如果需要显示群组名，需要额外查询
			// 目前先使用简化版本，类似 Telegram 的格式
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s mentioned you with a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s mentioned you", truncatedName)
		default:
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s mentioned you with a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s mentioned you", truncatedName)
		}
	}

	// 普通消息的内容
	switch msgType {
	case "private_chat":
		if userName != "" {
			truncatedName := pc.truncateUserName(userName)
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s sent you a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s sent you a message", truncatedName)
		}
		return "You have a new message"
	case "group_chat":
		if userName != "" {
			truncatedName := pc.truncateUserName(userName)
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s sent a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s sent a message", truncatedName)
		}
		return "New message in group"
	default:
		if userName != "" {
			truncatedName := pc.truncateUserName(userName)
			if chatInfoType == 1 || chatInfoType == 23 {
				return fmt.Sprintf("%s sent you a Candy Bag", truncatedName)
			}
			return fmt.Sprintf("%s sent you a message", truncatedName)
		}
		return "You have a new message"
	}
}

// truncateUserName 截取用户名，参考 Telegram 的处理方式
func (pc *PushCenter) truncateUserName(userName string) string {
	if userName == "" {
		return userName
	}

	// Telegram 通常将用户名限制在 20-25 个字符左右
	// 考虑到通知的显示空间，我们设置为 20 个字符
	const maxLength = 20

	if len(userName) <= maxLength {
		return userName
	}

	// 截取到 maxLength-3 个字符，然后添加 "..."
	// 这样总长度不会超过 maxLength
	truncated := userName[:maxLength-3] + "..."
	return truncated
}

// extractMessageContent 提取消息内容
func (pc *PushCenter) extractMessageContent(message interface{}) string {
	if message == nil {
		return ""
	}

	// 尝试转换为字符串
	if msgStr, ok := message.(string); ok {
		// 限制消息长度，避免推送内容过长
		if len(msgStr) > 100 {
			return msgStr[:100] + "..."
		}
		return msgStr
	}

	// 尝试解析为 JSON 并提取文本内容
	if msgMap, ok := message.(map[string]interface{}); ok {
		if text, exists := msgMap["text"]; exists {
			if textStr, ok := text.(string); ok {
				if len(textStr) > 100 {
					return textStr[:100] + "..."
				}
				return textStr
			}
		}
		if content, exists := msgMap["content"]; exists {
			if contentStr, ok := content.(string); ok {
				if len(contentStr) > 100 {
					return contentStr[:100] + "..."
				}
				return contentStr
			}
		}
	}

	// 尝试 JSON 序列化
	if jsonBytes, err := json.Marshal(message); err == nil {
		jsonStr := string(jsonBytes)
		if len(jsonStr) > 100 {
			return jsonStr[:100] + "..."
		}
		return jsonStr
	}

	return ""
}

// parseMessageInfo 解析 ExtraServiceMessage.Message 获取 pinId、groupId 和私聊的 metaId
func (pc *PushCenter) parseMessageInfo(chatMsg *socket_client_service.ChatNotificationMessage) (*ParsedMessageInfo, error) {
	if chatMsg == nil || chatMsg.Data == nil || chatMsg.Data.Message == nil {
		return nil, fmt.Errorf("聊天消息或消息内容为空")
	}

	parsedInfo := &ParsedMessageInfo{
		ChatType:     chatMsg.Type,
		PinId:        "", // 从 ExtraServiceMessage 直接获取 PinId
		ChatInfoType: 0,
	}

	// 尝试解析 Message 字段
	message := chatMsg.Data.Message

	// 方法1: 如果是 map 格式，直接解析
	if messageMap, ok := message.(map[string]interface{}); ok {
		// 解析 pinId（如果 Message 中有的话，会覆盖 ExtraServiceMessage 中的 PinId）
		if pinId, exists := messageMap["pinId"]; exists {
			if pinIdStr, ok := pinId.(string); ok {
				parsedInfo.PinId = pinIdStr
			}
		}

		// 解析 userInfo.name
		if userInfo, exists := messageMap["userInfo"]; exists {
			if userInfoMap, ok := userInfo.(map[string]interface{}); ok {
				if name, exists := userInfoMap["name"]; exists {
					if nameStr, ok := name.(string); ok {
						parsedInfo.UserName = nameStr
					}
				}
			}
		}

		// 根据聊天类型解析不同的字段
		switch chatMsg.Type {
		case "private_chat":
			// 私聊消息：解析 metaId（发送者或接收者）
			if metaId, exists := messageMap["metaId"]; exists {
				if metaIdStr, ok := metaId.(string); ok {
					parsedInfo.MetaId = metaIdStr
				}
			}
			// 如果没有 metaId，尝试从 from 或 to 字段获取
			if parsedInfo.MetaId == "" {
				if from, exists := messageMap["from"]; exists {
					if fromStr, ok := from.(string); ok {
						parsedInfo.MetaId = fromStr
					}
				}
			}
			if parsedInfo.MetaId == "" {
				if to, exists := messageMap["to"]; exists {
					if toStr, ok := to.(string); ok {
						parsedInfo.MetaId = toStr
					}
				}
			}

		case "group_chat":
			// 群聊消息：解析 groupId
			if groupId, exists := messageMap["groupId"]; exists {
				if groupIdStr, ok := groupId.(string); ok {
					parsedInfo.GroupId = groupIdStr
				}
			}
			// 如果没有 groupId，尝试从其他可能的字段获取
			if parsedInfo.GroupId == "" {
				if channelId, exists := messageMap["channelId"]; exists {
					if channelIdStr, ok := channelId.(string); ok {
						parsedInfo.GroupId = channelIdStr
					}
				}
			}

			fmt.Printf("messageMap: %+v\n", messageMap)
			if chatInfoType, exists := messageMap["chatType"]; exists {
				// 尝试多种数字类型转换
				switch v := chatInfoType.(type) {
				case int64:
					parsedInfo.ChatInfoType = v
				case int:
					parsedInfo.ChatInfoType = int64(v)
				case float64:
					parsedInfo.ChatInfoType = int64(v)
				case int32:
					parsedInfo.ChatInfoType = int64(v)
				case int16:
					parsedInfo.ChatInfoType = int64(v)
				case int8:
					parsedInfo.ChatInfoType = int64(v)
				default:
					log.Printf("⚠️ 无法转换 chatType 类型: %T, 值: %v", v, v)
				}
			}
		}

		log.Printf("📋 解析消息信息成功: PinId=%s, GroupId=%s, MetaId=%s, UserName=%s, ChatType=%s, ChatInfoType=%d",
			parsedInfo.PinId, parsedInfo.GroupId, parsedInfo.MetaId, parsedInfo.UserName, parsedInfo.ChatType, parsedInfo.ChatInfoType)
		return parsedInfo, nil
	}

	// // 方法2: 如果是字符串，尝试 JSON 解析
	// if messageStr, ok := message.(string); ok {
	// 	var messageMap map[string]interface{}
	// 	if err := json.Unmarshal([]byte(messageStr), &messageMap); err == nil {
	// 		// 递归调用，使用解析后的 map
	// 		tempChatMsg := &socket_client_service.ChatNotificationMessage{
	// 			Type: chatMsg.Type,
	// 			Data: &socket_client_service.ExtraServiceMessage{
	// 				Message:       messageMap,
	// 				RepostMetaIds: chatMsg.Data.RepostMetaIds,
	// 			},
	// 		}
	// 		return pc.parseMessageInfo(tempChatMsg)
	// 	}
	// }

	// 方法3: 如果无法解析，返回基本信息
	log.Printf("⚠️ 无法解析消息内容，使用基本信息: PinId=%s, ChatType=%s", parsedInfo.PinId, parsedInfo.ChatType)
	return parsedInfo, nil
}

// filterBlockedUsers 过滤掉已屏蔽该聊天的用户
func (pc *PushCenter) filterBlockedUsers(metaIds []string, parsedInfo *ParsedMessageInfo) []string {
	if len(metaIds) == 0 {
		return metaIds
	}

	var filteredMetaIds []string
	blockedCount := 0

	for _, metaId := range metaIds {
		// 确定要检查的聊天ID
		var chatID string
		if parsedInfo.ChatType == "private_chat" {
			// 私聊：使用私聊的metaId作为聊天ID
			chatID = parsedInfo.MetaId
			if metaId == chatID {
				// 自己不用给自己推送
				continue
			}
		} else if parsedInfo.ChatType == "group_chat" {
			// 群聊：使用groupId作为聊天ID
			chatID = parsedInfo.GroupId
		}

		// 如果没有聊天ID，跳过屏蔽检查
		if chatID == "" {
			filteredMetaIds = append(filteredMetaIds, metaId)
			continue
		}

		// 检查用户是否屏蔽了该聊天
		isBlocked, err := pebble_service.IsUserBlockedChat(metaId, chatID)
		if err != nil {
			log.Printf("⚠️ 检查用户 %s 屏蔽状态失败: %v，默认不屏蔽", metaId, err)
			// 出错时默认不屏蔽，继续推送
			filteredMetaIds = append(filteredMetaIds, metaId)
			continue
		}

		if isBlocked {
			blockedCount++
			log.Printf("🚫 用户 %s 已屏蔽聊天 %s，跳过推送", metaId, chatID)
		} else {
			filteredMetaIds = append(filteredMetaIds, metaId)
		}
	}

	if blockedCount > 0 {
		log.Printf("📊 屏蔽统计: %d 个用户已屏蔽该聊天", blockedCount)
	}

	return filteredMetaIds
}
