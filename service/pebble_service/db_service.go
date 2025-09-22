package pebble_service

import (
	"fmt"
	"push-base-service/models"
)

// SetUserToken 设置用户推送令牌（Token作为设备ID）
func SetUserToken(metaID, platform, token string) error {
	return SetUserPushToken(metaID, platform, token)
}

// GetUserTokenByMetaID 根据 metaId 获取用户推送令牌
func GetUserTokenByMetaID(metaID string) (*models.UserPushTokens, error) {
	if metaID == "" {
		return nil, fmt.Errorf("MetaID 不能为空")
	}

	return GetUserPushTokens(metaID)
}

// GetUserTokensList 获取用户推送令牌列表（支持分页）
func GetUserTokensList(page, pageSize int) (*PaginatedUserTokens, error) {
	return GetUserTokensListGlobal(page, pageSize)
}

// RemoveUserToken 移除用户指定平台的推送令牌
func RemoveUserToken(metaID, platform string) error {
	if metaID == "" {
		return fmt.Errorf("MetaID 不能为空")
	}
	if platform == "" {
		return fmt.Errorf("平台不能为空")
	}

	return RemoveUserPushToken(metaID, platform)
}

// RemoveUserAllTokens 移除用户的所有推送令牌
func RemoveUserAllTokens(metaID string) error {
	if metaID == "" {
		return fmt.Errorf("MetaID 不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.DeleteUserTokens(metaID)
}

// SetUserTokenWithDevice 设置用户推送令牌，同时管理设备信息
// 注意：deviceID 参数被忽略，因为现在使用 token 作为设备ID
func SetUserTokenWithDevice(metaID, platform, token, deviceID string) error {
	if metaID == "" {
		return fmt.Errorf("MetaID 不能为空")
	}
	if platform == "" {
		return fmt.Errorf("平台不能为空")
	}
	if token == "" {
		return fmt.Errorf("令牌不能为空")
	}
	// deviceID 参数被忽略，直接使用 SetUserToken
	return SetUserToken(metaID, platform, token)
}

// GetDeviceInfo 获取设备信息
func GetDeviceInfo(deviceID string) (*models.DeviceInfo, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("设备ID不能为空")
	}

	return GetDeviceInfoGlobal(deviceID)
}

// SetDeviceInfo 设置设备信息
func SetDeviceInfo(deviceID, platform, metaID string) error {
	if deviceID == "" {
		return fmt.Errorf("设备ID不能为空")
	}
	if platform == "" {
		return fmt.Errorf("平台不能为空")
	}
	if metaID == "" {
		return fmt.Errorf("MetaID不能为空")
	}

	return SetDeviceInfoGlobal(deviceID, platform, metaID)
}

// DeleteDeviceInfo 删除设备信息
func DeleteDeviceInfo(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("设备ID不能为空")
	}

	return DeleteDeviceInfoGlobal(deviceID)
}

// ===== 屏蔽聊天相关方法 =====

// GetUserBlockedChats 根据metaId获取用户屏蔽列表
func GetUserBlockedChats(metaID string) (*models.UserBlockedChats, error) {
	if metaID == "" {
		return nil, fmt.Errorf("MetaID不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.GetUserBlockedChats(metaID)
}

// AddBlockedChat 新增屏蔽某个群或某个私聊
func AddBlockedChat(metaID, chatID, chatType, reason string) error {
	if metaID == "" {
		return fmt.Errorf("MetaID不能为空")
	}
	if chatID == "" {
		return fmt.Errorf("ChatID不能为空")
	}
	if chatType == "" {
		return fmt.Errorf("ChatType不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.AddBlockedChat(metaID, chatID, chatType, reason)
}

// RemoveBlockedChat 取消屏蔽某个群或某个私聊
func RemoveBlockedChat(metaID, chatID string) error {
	if metaID == "" {
		return fmt.Errorf("MetaID不能为空")
	}
	if chatID == "" {
		return fmt.Errorf("ChatID不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.RemoveBlockedChat(metaID, chatID)
}

// IsUserBlockedChat 检查用户是否屏蔽了某个聊天（群聊或私聊）
func IsUserBlockedChat(metaID, chatID string) (bool, error) {
	if metaID == "" {
		return false, fmt.Errorf("MetaID不能为空")
	}
	if chatID == "" {
		return false, fmt.Errorf("ChatID不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return false, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return false, fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.IsBlockedChat(metaID, chatID)
}

// ===== PIN通知相关方法 =====

// AddNotifiedPin 添加PIN已通知记录
func AddNotifiedPin(pinID string) error {
	if pinID == "" {
		return fmt.Errorf("PinID不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.AddNotifiedPin(pinID)
}

// IsNotifiedPin 根据pinID获取是否已通知
func IsNotifiedPin(pinID string) (bool, error) {
	if pinID == "" {
		return false, fmt.Errorf("PinID不能为空")
	}

	service := GetGlobalService()
	if service == nil {
		return false, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}

	if !service.IsInitialized() {
		return false, fmt.Errorf("Pebble 服务未正确初始化")
	}

	return service.IsNotifiedPin(pinID)
}
