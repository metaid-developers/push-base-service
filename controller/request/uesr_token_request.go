package request

// SetUserTokensReq 设置用户推送令牌请求参数
type SetUserTokensReq struct {
	MetaID   string `json:"metaId" binding:"required"`
	Platform string `json:"platform" binding:"required"`
	Token    string `json:"token" binding:"required"` // Token本身就是设备的唯一标识
}

// GetUserTokenByMetaIDReq 根据 metaId 获取用户令牌请求参数
type GetUserTokenByMetaIDReq struct {
	MetaID string `json:"metaId" binding:"required"`
}

// GetUserTokensListReq 获取用户令牌列表请求参数（分页）
type GetUserTokensListReq struct {
	Page     int `json:"page" binding:"min=1"`     // 页码，从1开始
	PageSize int `json:"pageSize" binding:"min=1"` // 每页大小
}

// RemoveUserTokenReq 移除用户推送令牌请求参数
type RemoveUserTokenReq struct {
	MetaID   string `json:"metaId" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

// RemoveUserAllTokensReq 移除用户所有推送令牌请求参数
type RemoveUserAllTokensReq struct {
	MetaID string `json:"metaId" binding:"required"`
}

// ===== 屏蔽聊天相关请求参数 =====

// GetUserBlockedChatsReq 获取用户屏蔽聊天列表请求参数
type GetUserBlockedChatsReq struct {
	MetaID string `json:"metaId" binding:"required"`
}

// AddBlockedChatReq 添加屏蔽聊天请求参数
type AddBlockedChatReq struct {
	MetaID   string `json:"metaId" binding:"required"`
	ChatID   string `json:"chatId" binding:"required"`
	ChatType string `json:"chatType" binding:"required"` // 聊天类型：group, private
	Reason   string `json:"reason"`                      // 屏蔽原因（可选）
}

// RemoveBlockedChatReq 移除屏蔽聊天请求参数
type RemoveBlockedChatReq struct {
	MetaID string `json:"metaId" binding:"required"`
	ChatID string `json:"chatId" binding:"required"`
}
