package models

type UserPushTokens struct {
	MetaID    string            `json:"metaId" binding:"required"` // 用户唯一标识
	Tokens    map[string]string `json:"tokens"`                    // 平台->令牌映射 {"expo": "ExponentPushToken[...]", "fcm": "fcm_token_123"}
	UpdatedAt int64             `json:"updatedAt"`                 // 最后更新时间
}

// DeviceInfo 设备信息结构
type DeviceInfo struct {
	DeviceID  string `json:"deviceId" binding:"required"` // 设备唯一标识
	Platform  string `json:"platform" binding:"required"` // 平台 (expo, fcm, apns)
	MetaID    string `json:"metaId" binding:"required"`   // 关联的用户ID
	UpdatedAt int64  `json:"updatedAt"`                   // 最后更新时间
}

// BlockedChat 屏蔽聊天信息结构
type BlockedChat struct {
	UserID    string `json:"userId" binding:"required"` // 用户ID
	ChatID    string `json:"chatId" binding:"required"` // 群ID或私聊ID
	ChatType  string `json:"chatType"`                  // 聊天类型 (group, private)
	BlockedAt int64  `json:"blockedAt"`                 // 屏蔽时间
	Reason    string `json:"reason"`                    // 屏蔽原因
}

// UserBlockedChats 用户屏蔽聊天列表结构
type UserBlockedChats struct {
	UserID       string        `json:"userId" binding:"required"` // 用户ID
	BlockedChats []BlockedChat `json:"blockedChats"`              // 屏蔽的聊天列表
	UpdatedAt    int64         `json:"updatedAt"`                 // 最后更新时间
}

// NotifiedPin 已通知的PIN信息结构
type NotifiedPin struct {
	PinID       string `json:"pinId" binding:"required"` // PIN唯一标识
	ChatID      string `json:"chatId"`                   // 所属聊天ID
	UserID      string `json:"userId"`                   // 创建PIN的用户ID
	NotifiedAt  int64  `json:"notifiedAt"`               // 通知时间
	MessageHash string `json:"messageHash"`              // 消息哈希（用于去重）
}
