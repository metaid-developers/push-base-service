package socket_client_service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/zishang520/socket.io/clients/engine/v3/transports"
	socketio "github.com/zishang520/socket.io/clients/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

// Config Socket.IO 客户端配置
type Config struct {
	ServerURL        string `yaml:"server_url" json:"server_url"`                   // 服务器地址
	ExtraPushAuthKey string `yaml:"extra_push_auth_key" json:"extra_push_auth_key"` // 用户MetaID
	Path             string `yaml:"path" json:"path"`                               // Socket.IO路径，默认 "/socket.io/"
	Timeout          int    `yaml:"timeout" json:"timeout"`                         // 连接超时秒数，默认10秒
}

// SocketData WebSocket generic data structure
type SocketData struct {
	M string      `json:"M"`           // method
	C interface{} `json:"C"`           // code
	D interface{} `json:"D,omitempty"` // data
}

// PushMessage 推送消息
type PushMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// PushMessage 推送消息
type ChatNotificationMessage struct {
	Type string               `json:"type"`
	Data *ExtraServiceMessage `json:"data"`
}

// ExtraChatMessage 聊天消息
type ExtraServiceMessage struct {
	Message              interface{} `json:"message"`
	RepostMetaIds        []string    `json:"repostMetaIds"`
	MentionMetaIds       []string    `json:"mentionMetaIds"`
	RepostGlobalMetaIds  []string    `json:"repostGlobalMetaIds"`
	MentionGlobalMetaIds []string    `json:"mentionGlobalMetaIds"`
	// PinId         string      `json:"pinId"`
}

// WebSocket method constants
const (
	// Heartbeat
	HEART_BEAT                    = "HEART_BEAT"
	PONG                          = "PONG"
	WS_SERVER_NOTIFY_PRIVATE_CHAT = "WS_SERVER_NOTIFY_PRIVATE_CHAT"
	WS_SERVER_NOTIFY_GROUP_CHAT   = "WS_SERVER_NOTIFY_GROUP_CHAT"
	WS_SERVER_NOTIFY_GROUP_ROLE   = "WS_SERVER_NOTIFY_GROUP_ROLE"

	// Generic response
	WS_RESPONSE_SUCCESS = "WS_RESPONSE_SUCCESS"
	WS_RESPONSE_ERROR   = "WS_RESPONSE_ERROR"
)

// WebSocket code constants
const (
	WS_CODE_HEART_BEAT      = 10
	WS_CODE_HEART_BEAT_BACK = 10
	WS_CODE_SERVER          = 0
	WS_CODE_SEND_SUCCESS    = 200
	WS_CODE_SEND_ERROR      = 400
)

// Client Socket.IO 客户端
type Client struct {
	config    *Config
	socket    *socketio.Socket
	connected bool
	mu        sync.RWMutex

	// 消息处理回调
	OnMessage                 func(*PushMessage)
	OnChatNotificationMessage func(*ChatNotificationMessage) // 聊天消息回调
	OnHeartbeat               func()                         // 心跳回调
	OnConnect                 func()
	OnDisconnect              func()
	OnError                   func(error)
}

// NewClient 创建新的客户端
func NewClient(config *Config) *Client {
	if config.Path == "" {
		config.Path = "/socket.io/"
	}
	if config.Timeout == 0 {
		config.Timeout = 10
	}

	return &Client{
		config: config,
	}
}

// Start 启动客户端连接
func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket != nil && c.connected {
		return nil
	}

	// 构建连接URL，添加metaid参数
	serverURL := c.config.ServerURL

	// 创建Socket.IO连接选项
	options := socketio.DefaultOptions()
	options.SetTransports(types.NewSet(
		transports.Polling,
		transports.WebSocket,
	))
	options.SetPath(c.config.Path)
	options.SetQuery(
		url.Values{
			"extraPushAuthKey": {c.config.ExtraPushAuthKey},
		},
	)
	options.SetTimeout(time.Duration(c.config.Timeout) * time.Second)

	// 连接到服务器
	socket, err := socketio.Connect(serverURL, options)
	if err != nil {
		log.Printf("❌ Failed to connect to Socket.IO server: %v", err)
		if c.OnError != nil {
			go c.OnError(err)
		}
		return err
	}

	c.socket = socket

	// 设置事件处理器
	c.setupEventHandlers()

	log.Printf("🚀 Socket.IO client connecting to %s with ExtraPushAuthKey: %s", serverURL, c.config.ExtraPushAuthKey)

	return nil
}

// Stop 停止客户端
func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.socket != nil {
		c.socket.Disconnect()
		c.socket = nil
	}

	c.connected = false

	if c.OnDisconnect != nil {
		go c.OnDisconnect()
	}

	log.Println("📴 Socket.IO client stopped")
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.socket == nil {
		return false
	}

	// 安全地检查连接状态，防止 panic
	connected := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered when checking socket.Connected(): %v", r)
				connected = false
			}
		}()
		connected = c.socket.Connected()
	}()

	return connected
}

// setupEventHandlers 设置事件处理器
func (c *Client) setupEventHandlers() {
	if c.socket == nil {
		return
	}

	// 连接成功事件
	c.socket.On("connect", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in connect handler: %v", r)
			}
		}()

		c.mu.Lock()
		c.connected = true
		c.mu.Unlock()

		log.Printf("✅ Socket.IO connected successfully")

		if c.OnConnect != nil {
			go c.OnConnect()
		}

		// 启动心跳
		go c.startHeartbeat()
	})

	// 断开连接事件
	c.socket.On("disconnect", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in disconnect handler: %v", r)
			}
		}()

		c.mu.Lock()
		c.connected = false
		c.mu.Unlock()

		log.Printf("❌ Socket.IO disconnected")

		if c.OnDisconnect != nil {
			go c.OnDisconnect()
		}
	})

	// 连接错误事件
	c.socket.On("connect_error", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in connect_error handler: %v", r)
				// 即使 panic 了，也尝试通知错误处理器
				if c.OnError != nil {
					go c.OnError(fmt.Errorf("connect error panic recovered: %v", r))
				}
			}
		}()

		var err error
		if len(data) > 0 && data[0] != nil {
			if e, ok := data[0].(error); ok {
				err = e
			} else {
				// 如果不是 error 类型，创建一个错误
				err = fmt.Errorf("connection error: %v", data[0])
			}
		} else {
			// 如果没有错误数据，创建一个通用错误
			err = errors.New("connection error: unknown error")
		}

		log.Printf("🔥 Socket.IO connect error: %v", err)

		if c.OnError != nil {
			go c.OnError(err)
		}
	})

	// 通用错误事件（捕获其他类型的错误）
	c.socket.On("error", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in error handler: %v", r)
				if c.OnError != nil {
					go c.OnError(fmt.Errorf("error handler panic recovered: %v", r))
				}
			}
		}()

		var err error
		if len(data) > 0 && data[0] != nil {
			if e, ok := data[0].(error); ok {
				err = e
			} else {
				err = fmt.Errorf("socket error: %v", data[0])
			}
		} else {
			err = errors.New("socket error: unknown error")
		}

		log.Printf("🔥 Socket.IO error: %v", err)

		if c.OnError != nil {
			go c.OnError(err)
		}
	})

	// 处理服务端的WebSocket消息格式
	c.socket.On("message", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in message handler: %v", r)
			}
		}()

		c.handleSocketData(data)
	})

	// 兼容标准Socket.IO事件
	c.socket.On("push_message", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in push_message handler: %v", r)
			}
		}()

		c.handlePushMessage(data, "push_message")
	})

	c.socket.On("push_notification", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in push_notification handler: %v", r)
			}
		}()

		c.handlePushMessage(data, "push_notification")
	})

	c.socket.On("system_message", func(data ...interface{}) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("⚠️ Panic recovered in system_message handler: %v", r)
			}
		}()

		c.handlePushMessage(data, "system_message")
	})
}

// handlePushMessage 处理推送消息
func (c *Client) handlePushMessage(data []interface{}, eventType string) {
	if c.OnMessage == nil || len(data) == 0 {
		return
	}

	message := &PushMessage{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
	log.Printf("📨 Received %s: [%s] %s", eventType, message.Type, message.Data)

	// 异步调用消息处理器
	go c.OnMessage(message)
}

// handleSocketData 处理服务端的SocketData格式消息
func (c *Client) handleSocketData(data []interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Panic recovered in handleSocketData: %v", r)
		}
	}()

	if len(data) == 0 {
		return
	}

	// 尝试解析为SocketData格式
	var socketData *SocketData

	// 如果是字符串，直接解析
	if msgStr, ok := data[0].(string); ok {
		socketData = &SocketData{}
		err := json.Unmarshal([]byte(msgStr), socketData)
		if err != nil {
			log.Printf("⚠️ Failed to parse SocketData from string: %v", err)
			return
		}
	} else if msgMap, ok := data[0].(map[string]interface{}); ok {
		// 如果是map，转换为SocketData
		socketData = &SocketData{}
		if m, ok := msgMap["M"].(string); ok {
			socketData.M = m
		}
		if c, ok := msgMap["C"]; ok {
			socketData.C = c
		}
		if d, ok := msgMap["D"]; ok {
			socketData.D = d
		}
	} else {
		log.Printf("⚠️ Unknown SocketData format: %v", data[0])
		return
	}

	log.Printf("📡 Received SocketData: M=%s, C=%v", socketData.M, socketData.C)

	// 根据方法类型处理消息
	switch strings.ToUpper(socketData.M) {
	case HEART_BEAT, PONG:
		c.handleHeartbeatMessage(socketData)
	case WS_SERVER_NOTIFY_PRIVATE_CHAT:
		c.handlePrivateChatMessage(socketData)
	case WS_SERVER_NOTIFY_GROUP_CHAT, WS_SERVER_NOTIFY_GROUP_ROLE:
		c.handleGroupChatMessage(socketData)
	default:
		log.Printf("📨 未知方法: %s, 数据: %v", socketData.M, socketData.D)
	}
}

// handleHeartbeatMessage 处理心跳消息
func (c *Client) handleHeartbeatMessage(socketData *SocketData) {
	log.Printf("💓 收到服务端心跳: M=%s, C=%v, D=%v", socketData.M, socketData.C, socketData.D)

	if c.OnHeartbeat != nil {
		go c.OnHeartbeat()
	}
}

// handlePrivateChatMessage 处理私聊消息
func (c *Client) handlePrivateChatMessage(socketData *SocketData) {
	log.Printf("💬 收到私聊消息: %v", socketData.M)

	// 序列化 socketData.D 为 ExtraServiceMessage
	data, err := c.parseExtraServiceMessage(socketData.D)
	if err != nil {
		log.Printf("⚠️ 解析私聊消息失败: %v", err)
		return
	}

	if c.OnChatNotificationMessage != nil {
		chatMessage := &ChatNotificationMessage{
			Type: "private_chat",
			Data: data,
		}
		go c.OnChatNotificationMessage(chatMessage)
	}
}

// handleGroupChatMessage 处理群聊消息
func (c *Client) handleGroupChatMessage(socketData *SocketData) {
	log.Printf("👥 收到群聊消息: %v", socketData.M)

	// 序列化 socketData.D 为 ExtraServiceMessage
	data, err := c.parseExtraServiceMessage(socketData.D)
	if err != nil {
		log.Printf("⚠️ 解析群聊消息失败: %v", err)
		return
	}

	if c.OnChatNotificationMessage != nil {
		chatMessage := &ChatNotificationMessage{
			Type: "group_chat",
			Data: data,
		}
		go c.OnChatNotificationMessage(chatMessage)
	}
}

// parseExtraServiceMessage 解析 socketData.D 为 ExtraServiceMessage
func (c *Client) parseExtraServiceMessage(data interface{}) (*ExtraServiceMessage, error) {
	if data == nil {
		return &ExtraServiceMessage{
			Message:              nil,
			RepostMetaIds:        []string{},
			MentionMetaIds:       []string{},
			RepostGlobalMetaIds:  []string{},
			MentionGlobalMetaIds: []string{},
		}, nil
	}

	// 方法1: 如果是map格式，直接转换
	if dataMap, ok := data.(map[string]interface{}); ok {
		extraMsg := &ExtraServiceMessage{
			RepostMetaIds:        []string{},
			MentionMetaIds:       []string{},
			RepostGlobalMetaIds:  []string{},
			MentionGlobalMetaIds: []string{},
		}

		// 解析 message 字段
		if message, exists := dataMap["message"]; exists {
			extraMsg.Message = message
		} else {
			// 如果没有 message 字段，将整个 data 作为 message
			extraMsg.Message = data
		}

		// 解析 repostMetaIds 字段
		if repostIds, exists := dataMap["repostMetaIds"]; exists {
			if repostArray, ok := repostIds.([]interface{}); ok {
				for _, id := range repostArray {
					if idStr, ok := id.(string); ok {
						extraMsg.RepostMetaIds = append(extraMsg.RepostMetaIds, idStr)
					}
				}
			}
		}

		// 解析 mentionMetaIds 字段
		if mentionIds, exists := dataMap["mentionMetaIds"]; exists {
			if mentionArray, ok := mentionIds.([]interface{}); ok {
				for _, id := range mentionArray {
					if idStr, ok := id.(string); ok {
						extraMsg.MentionMetaIds = append(extraMsg.MentionMetaIds, idStr)
					}
				}
			}
		}

		// 解析 repostGlobalMetaIds 字段
		if repostGlobalIds, exists := dataMap["repostGlobalMetaIds"]; exists {
			if repostGlobalArray, ok := repostGlobalIds.([]interface{}); ok {
				for _, id := range repostGlobalArray {
					if idStr, ok := id.(string); ok {
						extraMsg.RepostGlobalMetaIds = append(extraMsg.RepostGlobalMetaIds, idStr)
					}
				}
			}
		}

		// 解析 mentionGlobalMetaIds 字段
		if mentionGlobalIds, exists := dataMap["mentionGlobalMetaIds"]; exists {
			if mentionGlobalArray, ok := mentionGlobalIds.([]interface{}); ok {
				for _, id := range mentionGlobalArray {
					if idStr, ok := id.(string); ok {
						extraMsg.MentionGlobalMetaIds = append(extraMsg.MentionGlobalMetaIds, idStr)
					}
				}
			}
		}
		// // 解析 pinId 字段
		// if pinId, exists := dataMap["pinId"]; exists {
		// 	if pinIdStr, ok := pinId.(string); ok {
		// 		extraMsg.PinId = pinIdStr
		// 	}
		// }

		return extraMsg, nil
	}

	// 方法2: 如果是字符串，尝试JSON解析
	if dataStr, ok := data.(string); ok {
		extraMsg := &ExtraServiceMessage{}
		err := json.Unmarshal([]byte(dataStr), extraMsg)
		if err != nil {
			// 如果JSON解析失败，将字符串作为message
			return &ExtraServiceMessage{
				Message:              dataStr,
				RepostMetaIds:        []string{},
				MentionMetaIds:       []string{},
				RepostGlobalMetaIds:  []string{},
				MentionGlobalMetaIds: []string{},
			}, nil
		}
		return extraMsg, nil
	}

	// 方法3: 其他类型，直接作为message
	return &ExtraServiceMessage{
		Message:              data,
		RepostMetaIds:        []string{},
		MentionMetaIds:       []string{},
		RepostGlobalMetaIds:  []string{},
		MentionGlobalMetaIds: []string{},
	}, nil
}

// sendSocketData 发送SocketData格式消息
func (c *Client) sendSocketData(socketData *SocketData) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Panic recovered in sendSocketData: %v", r)
		}
	}()

	c.mu.RLock()
	socket := c.socket
	c.mu.RUnlock()

	if socket == nil || !c.IsConnected() {
		return errors.New("client not connected")
	}

	// 发送SocketData格式的消息
	socket.Emit("message", socketData)
	return nil
}

// startHeartbeat 启动心跳
func (c *Client) startHeartbeat() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Panic recovered in startHeartbeat: %v", r)
		}
	}()

	ticker := time.NewTicker(5 * time.Second) // 每5秒发送心跳
	defer ticker.Stop()

	for range ticker.C {
		// 使用 recover 保护每次心跳发送
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("⚠️ Panic recovered in heartbeat tick: %v", r)
				}
			}()

			if c.IsConnected() {
				c.sendHeartbeat()
			} else {
				return // 连接断开，退出心跳
			}
		}()
	}
}

// sendHeartbeat 发送心跳
func (c *Client) sendHeartbeat() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Panic recovered in sendHeartbeat: %v", r)
		}
	}()

	c.mu.RLock()
	socket := c.socket
	c.mu.RUnlock()

	if socket == nil || !c.IsConnected() {
		return
	}

	// 使用SocketData格式发送心跳
	heartbeatData := &SocketData{
		M: PONG,
		C: WS_CODE_HEART_BEAT,
	}

	c.sendSocketData(heartbeatData)
	// log.Printf("❤️ Heartbeat sent (SocketData format)")
}

// SendMessage 发送自定义消息
func (c *Client) SendMessage(event string, data interface{}) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("⚠️ Panic recovered in SendMessage: %v", r)
		}
	}()

	c.mu.RLock()
	socket := c.socket
	c.mu.RUnlock()

	if socket == nil || !c.IsConnected() {
		log.Printf("❌ Client not connected")
		return errors.New("client not connected")
	}

	socket.Emit(event, data)
	log.Printf("📤 Sent event: %s", event)

	return nil
}
