package pebble_service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"push-base-service/models"
	"push-base-service/service/push_service"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/pebble"
)

var Pb map[string]*pebble.DB

const (
	CollectionUserTokens   = "user_tokens"   // 用户令牌集合
	CollectionDevices      = "devices"       // 设备信息集合
	CollectionBlockedChats = "blocked_chats" // 用户屏蔽的群ID或私聊ID集合 key:metaid, value: []{groupId or chatId, type}
	CollectionNotifiedPins = "notified_pins" // 已经通知的PIN ID集合 key: pinId, value: pinId
)

// PebbleService Pebble 数据库服务
type PebbleService struct {
	collectionMgr *CollectionManager // 集合管理器
	mu            sync.RWMutex
	path          string
}

// Config Pebble 配置
type Config struct {
	DBPath string `yaml:"db_path" json:"db_path"` // 数据库文件路径
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DBPath: "./data/pebble", // 默认数据库路径
	}
}

// CollectionManager 集合管理器
type CollectionManager struct {
	mu          sync.RWMutex
	collections map[string]*pebble.DB
	basePath    string
}

// NewCollectionManager 创建集合管理器
func NewCollectionManager(basePath string) *CollectionManager {
	return &CollectionManager{
		collections: make(map[string]*pebble.DB),
		basePath:    basePath,
	}
}

// GetCollection 获取指定集合的数据库实例
func (cm *CollectionManager) GetCollection(collectionName string) (*pebble.DB, error) {
	cm.mu.RLock()
	if db, exists := cm.collections[collectionName]; exists {
		cm.mu.RUnlock()
		return db, nil
	}
	cm.mu.RUnlock()

	// 需要创建新的数据库实例
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 双重检查，防止并发创建
	if db, exists := cm.collections[collectionName]; exists {
		return db, nil
	}

	// 创建集合专用的数据库路径
	dbPath := filepath.Join(cm.basePath, collectionName)

	// 配置 Pebble 选项
	opts := &pebble.Options{
		Cache:                       pebble.NewCache(16 << 20), // 16MB 缓存
		DisableWAL:                  false,                     // 启用 WAL
		FormatMajorVersion:          pebble.FormatNewest,       // 使用最新格式
		L0CompactionThreshold:       2,                         // L0 压缩阈值
		L0StopWritesThreshold:       1000,                      // L0 停止写入阈值
		LBaseMaxBytes:               16 << 20,                  // 16MB
		MaxOpenFiles:                4096,                      // 最大打开文件数
		MemTableSize:                16 << 20,                  // 16MB 内存表
		MemTableStopWritesThreshold: 4,                         // 内存表停止写入阈值
	}

	// 打开数据库
	db, err := pebble.Open(dbPath, opts)
	if err != nil {
		return nil, fmt.Errorf("打开集合 %s 的数据库失败: %w", collectionName, err)
	}

	cm.collections[collectionName] = db
	log.Printf("✅ 集合 %s 数据库初始化成功: %s", collectionName, dbPath)

	return db, nil
}

// CloseCollection 关闭指定集合的数据库
func (cm *CollectionManager) CloseCollection(collectionName string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if db, exists := cm.collections[collectionName]; exists {
		err := db.Close()
		delete(cm.collections, collectionName)
		if err != nil {
			return fmt.Errorf("关闭集合 %s 的数据库失败: %w", collectionName, err)
		}
		log.Printf("✅ 集合 %s 数据库已关闭", collectionName)
	}
	return nil
}

// CloseAll 关闭所有集合的数据库
func (cm *CollectionManager) CloseAll() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errors []string
	for collectionName, db := range cm.collections {
		if err := db.Close(); err != nil {
			errors = append(errors, fmt.Sprintf("关闭集合 %s 失败: %v", collectionName, err))
		} else {
			log.Printf("✅ 集合 %s 数据库已关闭", collectionName)
		}
	}

	cm.collections = make(map[string]*pebble.DB)

	if len(errors) > 0 {
		return fmt.Errorf("关闭数据库时发生错误: %s", strings.Join(errors, "; "))
	}
	return nil
}

// ListCollections 列出所有已初始化的集合
func (cm *CollectionManager) ListCollections() []string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var collections []string
	for name := range cm.collections {
		collections = append(collections, name)
	}
	return collections
}

// NewPebbleService 创建新的 Pebble 服务实例
func NewPebbleService(config *Config) *PebbleService {
	if config == nil {
		config = DefaultConfig()
	}

	return &PebbleService{
		path:          config.DBPath,
		collectionMgr: NewCollectionManager(config.DBPath),
	}
}

// Initialize 初始化 Pebble 数据库
func (ps *PebbleService) Initialize() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	log.Printf("🚀 正在初始化 Pebble 数据库: %s", ps.path)

	// 创建数据库目录
	dbPath, err := filepath.Abs(ps.path)
	if err != nil {
		return fmt.Errorf("获取数据库路径失败: %w", err)
	}

	log.Printf("✅ Pebble 数据库初始化成功: %s", dbPath)

	return nil
}

// Close 关闭数据库
func (ps *PebbleService) Close() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	log.Printf("🛑 正在关闭 Pebble 数据库")

	// 关闭所有集合数据库
	if ps.collectionMgr != nil {
		if err := ps.collectionMgr.CloseAll(); err != nil {
			log.Printf("❌ 关闭集合数据库失败: %v", err)
			return fmt.Errorf("关闭集合数据库失败: %w", err)
		}
	}

	log.Printf("✅ Pebble 数据库已关闭")
	return nil
}

// IsInitialized 检查数据库是否已初始化
func (ps *PebbleService) IsInitialized() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.collectionMgr != nil
}

// getCollectionDB 获取指定集合的数据库实例
func (ps *PebbleService) getCollectionDB(collectionName string) (*pebble.DB, error) {
	if ps.collectionMgr == nil {
		return nil, fmt.Errorf("集合管理器未初始化")
	}
	return ps.collectionMgr.GetCollection(collectionName)
}

// buildKey 构建集合键（现在每个collection都有独立的数据库，所以键可以简化）
func buildKey(id string) []byte {
	return []byte(id)
}

// getUserTokensKey 生成用户令牌的键
func getUserTokensKey(metaId string) []byte {
	return buildKey(metaId)
}

// getDeviceKey 生成设备ID的键
func getDeviceKey(deviceId string) []byte {
	return buildKey(deviceId)
}

// getUserBlockedChatsKey 生成用户屏蔽聊天列表的键
func getUserBlockedChatsKey(userId string) []byte {
	return buildKey(userId)
}

// getNotifiedPinKey 生成已通知PIN的键
func getNotifiedPinKey(pinId string) []byte {
	return buildKey(pinId)
}

// getUserBlockedChatsFromDB 从数据库获取用户屏蔽聊天列表
func (ps *PebbleService) getUserBlockedChatsFromDB(db *pebble.DB, userId string) (*models.UserBlockedChats, error) {
	key := getUserBlockedChatsKey(userId)
	value, closer, err := db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			// 用户没有屏蔽列表，返回空列表
			return &models.UserBlockedChats{
				UserID:       userId,
				BlockedChats: []models.BlockedChat{},
				UpdatedAt:    time.Now().Unix(),
			}, nil
		}
		return nil, fmt.Errorf("获取用户屏蔽列表失败: %w", err)
	}
	defer closer.Close()

	// 反序列化 JSON
	var userBlockedChats models.UserBlockedChats
	if err := json.Unmarshal(value, &userBlockedChats); err != nil {
		return nil, fmt.Errorf("反序列化用户屏蔽列表失败: %w", err)
	}

	return &userBlockedChats, nil
}

// SaveUserTokens 保存用户推送令牌
func (ps *PebbleService) SaveUserTokens(userTokens *models.UserPushTokens) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if userTokens.MetaID == "" {
		return fmt.Errorf("MetaID 不能为空")
	}

	// 获取用户令牌集合的数据库
	db, err := ps.getCollectionDB(CollectionUserTokens)
	if err != nil {
		return fmt.Errorf("获取用户令牌集合数据库失败: %w", err)
	}

	// 设置更新时间
	userTokens.UpdatedAt = time.Now().Unix()

	// 序列化为 JSON
	data, err := json.Marshal(userTokens)
	if err != nil {
		return fmt.Errorf("序列化用户令牌失败: %w", err)
	}

	// 保存到数据库
	key := getUserTokensKey(userTokens.MetaID)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return fmt.Errorf("保存用户令牌失败: %w", err)
	}

	log.Printf("✅ 已保存用户令牌: MetaID=%s, 平台数=%d", userTokens.MetaID, len(userTokens.Tokens))
	return nil
}

// GetUserTokens 获取用户推送令牌
func (ps *PebbleService) GetUserTokens(metaId string) (*models.UserPushTokens, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if metaId == "" {
		return nil, fmt.Errorf("MetaID 不能为空")
	}

	// 获取用户令牌集合的数据库
	db, err := ps.getCollectionDB(CollectionUserTokens)
	if err != nil {
		return nil, fmt.Errorf("获取用户令牌集合数据库失败: %w", err)
	}

	key := getUserTokensKey(metaId)
	value, closer, err := db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			// 用户不存在，返回空的令牌结构
			return &models.UserPushTokens{
				MetaID:    metaId,
				Tokens:    make(map[string]string),
				UpdatedAt: time.Now().Unix(),
			}, nil
		}
		return nil, fmt.Errorf("获取用户令牌失败: %w", err)
	}
	defer closer.Close()

	// 反序列化 JSON
	var userTokens models.UserPushTokens
	if err := json.Unmarshal(value, &userTokens); err != nil {
		return nil, fmt.Errorf("反序列化用户令牌失败: %w", err)
	}

	log.Printf("📖 已获取用户令牌: MetaID=%s, 平台数=%d", userTokens.MetaID, len(userTokens.Tokens))
	return &userTokens, nil
}

// UpdateUserTokens 更新用户推送令牌
func (ps *PebbleService) UpdateUserTokens(userTokens *models.UserPushTokens) error {
	// 更新操作与保存操作相同
	return ps.SaveUserTokens(userTokens)
}

// SetUserToken 设置用户在指定平台的推送令牌（Token作为设备ID进行唯一性检查）
func (ps *PebbleService) SetUserToken(metaId, platform, token string) error {
	if metaId == "" || platform == "" || token == "" {
		return fmt.Errorf("MetaID、平台和令牌都不能为空")
	}

	// 1. 使用token作为设备ID，检查是否已存在，如果存在且属于不同用户，需要处理冲突
	existingDevice, err := ps.GetDeviceInfo(token) // 使用token作为deviceId
	if err == nil {
		// 设备(token)已存在
		if existingDevice.MetaID != metaId {
			// Token属于不同用户，需要从旧用户中移除该平台的令牌
			log.Printf("⚠️ Token %s 从用户 %s 转移到用户 %s", token, existingDevice.MetaID, metaId)

			// 获取旧用户的令牌
			oldUserTokens, err := ps.GetUserTokens(existingDevice.MetaID)
			if err == nil && oldUserTokens.Tokens != nil {
				// 从旧用户的令牌中移除该平台的令牌
				if oldToken, exists := oldUserTokens.Tokens[platform]; exists && oldToken == token {
					delete(oldUserTokens.Tokens, platform)
					if err := ps.SaveUserTokens(oldUserTokens); err != nil {
						log.Printf("⚠️ 更新旧用户 %s 令牌失败: %v", existingDevice.MetaID, err)
					} else {
						log.Printf("✅ 已从旧用户 %s 中移除平台 %s 的令牌", existingDevice.MetaID, platform)
					}
				}
			}
		}
		// 更新设备信息到新用户
		existingDevice.MetaID = metaId
		existingDevice.Platform = platform
		if err := ps.SaveDeviceInfo(existingDevice); err != nil {
			return fmt.Errorf("更新设备信息失败: %w", err)
		}
	} else {
		// Token(设备)不存在，创建新的设备信息
		deviceInfo := &models.DeviceInfo{
			DeviceID:  token, // 使用token作为设备ID
			Platform:  platform,
			MetaID:    metaId,
			UpdatedAt: time.Now().Unix(),
		}
		if err := ps.SaveDeviceInfo(deviceInfo); err != nil {
			return fmt.Errorf("创建设备信息失败: %w", err)
		}
	}

	// 2. 获取现有用户令牌
	userTokens, err := ps.GetUserTokens(metaId)
	if err != nil {
		return fmt.Errorf("获取现有用户令牌失败: %w", err)
	}

	// 确保 Tokens map 不为 nil
	if userTokens.Tokens == nil {
		userTokens.Tokens = make(map[string]string)
	}

	// 3. 设置令牌
	userTokens.Tokens[platform] = token

	// 4. 保存更新后的令牌
	if err := ps.SaveUserTokens(userTokens); err != nil {
		return fmt.Errorf("保存更新后的用户令牌失败: %w", err)
	}

	log.Printf("✅ 已设置用户令牌: MetaID=%s, 平台=%s, Token(DeviceID)=%s", metaId, platform, token)
	return nil
}

// SetUserTokenWithDevice 设置用户在指定平台的推送令牌，同时管理设备信息
// 注意：此方法现在直接调用 SetUserToken，因为 SetUserToken 已经使用token作为设备ID
func (ps *PebbleService) SetUserTokenWithDevice(metaId, platform, token, deviceId string) error {
	// deviceId 参数被忽略，因为我们使用 token 作为设备ID
	// 直接调用 SetUserToken，因为它现在已经包含了完整的设备管理逻辑
	return ps.SetUserToken(metaId, platform, token)
}

// RemoveUserToken 移除用户在指定平台的推送令牌
func (ps *PebbleService) RemoveUserToken(metaId, platform string) error {
	if metaId == "" || platform == "" {
		return fmt.Errorf("MetaID 和平台不能为空")
	}

	// 获取现有令牌
	userTokens, err := ps.GetUserTokens(metaId)
	if err != nil {
		return fmt.Errorf("获取现有用户令牌失败: %w", err)
	}

	// 确保 Tokens map 不为 nil
	if userTokens.Tokens == nil {
		log.Printf("⚠️ 用户 %s 没有令牌记录", metaId)
		return nil
	}

	// 检查令牌是否存在
	if _, exists := userTokens.Tokens[platform]; !exists {
		log.Printf("⚠️ 用户 %s 在平台 %s 上没有令牌", metaId, platform)
		return nil
	}

	// 移除令牌
	delete(userTokens.Tokens, platform)

	// 保存更新后的令牌
	if err := ps.SaveUserTokens(userTokens); err != nil {
		return fmt.Errorf("保存更新后的用户令牌失败: %w", err)
	}

	log.Printf("✅ 已移除用户令牌: MetaID=%s, 平台=%s", metaId, platform)
	return nil
}

// DeleteUserTokens 删除用户的所有推送令牌
func (ps *PebbleService) DeleteUserTokens(metaId string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if metaId == "" {
		return fmt.Errorf("MetaID 不能为空")
	}

	// 获取用户令牌集合的数据库
	db, err := ps.getCollectionDB(CollectionUserTokens)
	if err != nil {
		return fmt.Errorf("获取用户令牌集合数据库失败: %w", err)
	}

	key := getUserTokensKey(metaId)
	if err := db.Delete(key, pebble.Sync); err != nil {
		return fmt.Errorf("删除用户令牌失败: %w", err)
	}

	log.Printf("🗑️ 已删除用户所有令牌: MetaID=%s", metaId)
	return nil
}

// SaveDeviceInfo 保存设备信息
func (ps *PebbleService) SaveDeviceInfo(deviceInfo *models.DeviceInfo) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if deviceInfo.DeviceID == "" {
		return fmt.Errorf("DeviceID 不能为空")
	}

	if deviceInfo.Platform == "" {
		return fmt.Errorf("Platform 不能为空")
	}

	if deviceInfo.MetaID == "" {
		return fmt.Errorf("MetaID 不能为空")
	}

	// 获取设备集合的数据库
	db, err := ps.getCollectionDB(CollectionDevices)
	if err != nil {
		return fmt.Errorf("获取设备集合数据库失败: %w", err)
	}

	// 设置更新时间
	deviceInfo.UpdatedAt = time.Now().Unix()

	// 序列化为 JSON
	data, err := json.Marshal(deviceInfo)
	if err != nil {
		return fmt.Errorf("序列化设备信息失败: %w", err)
	}

	// 保存到数据库
	key := getDeviceKey(deviceInfo.DeviceID)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return fmt.Errorf("保存设备信息失败: %w", err)
	}

	log.Printf("✅ 已保存设备信息: DeviceID=%s, Platform=%s, MetaID=%s",
		deviceInfo.DeviceID, deviceInfo.Platform, deviceInfo.MetaID)
	return nil
}

// GetDeviceInfo 获取设备信息
func (ps *PebbleService) GetDeviceInfo(deviceId string) (*models.DeviceInfo, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if deviceId == "" {
		return nil, fmt.Errorf("DeviceID 不能为空")
	}

	// 获取设备集合的数据库
	db, err := ps.getCollectionDB(CollectionDevices)
	if err != nil {
		return nil, fmt.Errorf("获取设备集合数据库失败: %w", err)
	}

	key := getDeviceKey(deviceId)
	value, closer, err := db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, fmt.Errorf("设备 %s 不存在", deviceId)
		}
		return nil, fmt.Errorf("获取设备信息失败: %w", err)
	}
	defer closer.Close()

	// 反序列化 JSON
	var deviceInfo models.DeviceInfo
	if err := json.Unmarshal(value, &deviceInfo); err != nil {
		return nil, fmt.Errorf("反序列化设备信息失败: %w", err)
	}

	log.Printf("📖 已获取设备信息: DeviceID=%s, Platform=%s, MetaID=%s",
		deviceInfo.DeviceID, deviceInfo.Platform, deviceInfo.MetaID)
	return &deviceInfo, nil
}

// UpdateDeviceInfo 更新设备信息
func (ps *PebbleService) UpdateDeviceInfo(deviceInfo *models.DeviceInfo) error {
	// 更新操作与保存操作相同
	return ps.SaveDeviceInfo(deviceInfo)
}

// DeleteDeviceInfo 删除设备信息
func (ps *PebbleService) DeleteDeviceInfo(deviceId string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if deviceId == "" {
		return fmt.Errorf("DeviceID 不能为空")
	}

	// 获取设备集合的数据库
	db, err := ps.getCollectionDB(CollectionDevices)
	if err != nil {
		return fmt.Errorf("获取设备集合数据库失败: %w", err)
	}

	key := getDeviceKey(deviceId)
	if err := db.Delete(key, pebble.Sync); err != nil {
		return fmt.Errorf("删除设备信息失败: %w", err)
	}

	log.Printf("🗑️ 已删除设备信息: DeviceID=%s", deviceId)
	return nil
}

// SetDeviceInfo 设置设备信息（如果设备已存在且MetaID不同，则更新）
func (ps *PebbleService) SetDeviceInfo(deviceId, platform, metaId string) error {
	if deviceId == "" || platform == "" || metaId == "" {
		return fmt.Errorf("DeviceID、Platform 和 MetaID 都不能为空")
	}

	// 检查设备是否已存在
	existingDevice, err := ps.GetDeviceInfo(deviceId)
	if err == nil {
		// 设备存在，检查是否需要更新
		if existingDevice.MetaID != metaId {
			log.Printf("⚠️ 设备 %s 的 MetaID 从 %s 更改为 %s", deviceId, existingDevice.MetaID, metaId)

			// 需要从旧用户的令牌中移除该设备的令牌
			oldUserTokens, err := ps.GetUserTokens(existingDevice.MetaID)
			if err == nil && oldUserTokens.Tokens != nil {
				// 移除旧用户在该平台的令牌（如果该令牌对应这个设备）
				if _, exists := oldUserTokens.Tokens[platform]; exists {
					delete(oldUserTokens.Tokens, platform)
					if err := ps.SaveUserTokens(oldUserTokens); err != nil {
						log.Printf("⚠️ 更新旧用户令牌失败: %v", err)
					} else {
						log.Printf("✅ 已从旧用户 %s 中移除平台 %s 的令牌", existingDevice.MetaID, platform)
					}
				}
			}
		}

		// 更新设备信息
		existingDevice.Platform = platform
		existingDevice.MetaID = metaId
		return ps.SaveDeviceInfo(existingDevice)
	}

	// 设备不存在，创建新的设备信息
	deviceInfo := &models.DeviceInfo{
		DeviceID:  deviceId,
		Platform:  platform,
		MetaID:    metaId,
		UpdatedAt: time.Now().Unix(),
	}

	return ps.SaveDeviceInfo(deviceInfo)
}

// GetAllUserTokens 获取多个用户的推送令牌
func (ps *PebbleService) GetAllUserTokens(metaIds []string) (map[string]*models.UserPushTokens, error) {
	if len(metaIds) == 0 {
		return make(map[string]*models.UserPushTokens), nil
	}

	result := make(map[string]*models.UserPushTokens)

	for _, metaId := range metaIds {
		userTokens, err := ps.GetUserTokens(metaId)
		if err != nil {
			log.Printf("⚠️ 获取用户 %s 的令牌失败: %v", metaId, err)
			// 创建空的令牌记录
			userTokens = &models.UserPushTokens{
				MetaID:    metaId,
				Tokens:    make(map[string]string),
				UpdatedAt: time.Now().Unix(),
			}
		}
		result[metaId] = userTokens
	}

	log.Printf("📖 已获取 %d 个用户的令牌", len(result))
	return result, nil
}

// Stats 获取数据库统计信息
func (ps *PebbleService) Stats() (map[string]interface{}, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.collectionMgr == nil {
		return nil, fmt.Errorf("集合管理器未初始化")
	}

	// 获取集合列表
	collections := ps.collectionMgr.ListCollections()

	stats := map[string]interface{}{
		"path":        ps.path,
		"initialized": true,
		"collections": collections,
	}

	return stats, nil
}

// 全局服务实例
var (
	globalService *PebbleService
	globalOnce    sync.Once
)

// GetGlobalService 获取全局 Pebble 服务实例
func GetGlobalService() *PebbleService {
	// 如果全局服务已存在，直接返回
	if globalService != nil {
		return globalService
	}

	// 全局服务不存在，抛出错误而不是创建新实例
	log.Printf("❌ 全局 Pebble 服务未初始化，请先调用 InitializeGlobalService")
	return nil
}

// InitializeGlobalService 初始化全局服务
func InitializeGlobalService(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	// 如果全局服务已存在且已初始化，直接返回
	if globalService != nil && globalService.IsInitialized() {
		log.Printf("⚠️ 全局 Pebble 服务已存在，跳过重复初始化")
		return nil
	}

	// 重置全局实例，确保使用新配置
	globalOnce = sync.Once{}

	service := NewPebbleService(config)
	if err := service.Initialize(); err != nil {
		return fmt.Errorf("初始化全局 Pebble 服务失败: %w", err)
	}

	globalService = service
	log.Printf("✅ 全局 Pebble 服务初始化完成: %s", config.DBPath)
	return nil
}

// CloseGlobalService 关闭全局服务
func CloseGlobalService() error {
	if globalService != nil {
		return globalService.Close()
	}
	return nil
}

// GetUserPushTokens 全局方法：获取用户推送令牌
func GetUserPushTokens(metaId string) (*models.UserPushTokens, error) {
	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.GetUserTokens(metaId)
}

// SetUserPushToken 全局方法：设置用户推送令牌（Token作为设备ID进行唯一性检查）
func SetUserPushToken(metaId, platform, token string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.SetUserToken(metaId, platform, token)
}

// RemoveUserPushToken 全局方法：移除用户推送令牌
func RemoveUserPushToken(metaId, platform string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.RemoveUserToken(metaId, platform)
}

// GetAllUserPushTokens 全局方法：批量获取用户推送令牌
func GetAllUserPushTokens(metaIds []string) (map[string]*models.UserPushTokens, error) {
	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.GetAllUserTokens(metaIds)
}

// SetUserTokenWithDeviceGlobal 全局方法：设置用户推送令牌和设备信息
func SetUserTokenWithDeviceGlobal(metaId, platform, token, deviceId string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.SetUserTokenWithDevice(metaId, platform, token, deviceId)
}

// GetDeviceInfoGlobal 全局方法：获取设备信息
func GetDeviceInfoGlobal(deviceId string) (*models.DeviceInfo, error) {
	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.GetDeviceInfo(deviceId)
}

// SetDeviceInfoGlobal 全局方法：设置设备信息
func SetDeviceInfoGlobal(deviceId, platform, metaId string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.SetDeviceInfo(deviceId, platform, metaId)
}

// DeleteDeviceInfoGlobal 全局方法：删除设备信息
func DeleteDeviceInfoGlobal(deviceId string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.DeleteDeviceInfo(deviceId)
}

// PaginatedUserTokens 分页的用户令牌结果
type PaginatedUserTokens struct {
	Users      []*models.UserPushTokens `json:"users"`      // 用户令牌列表
	Total      int                      `json:"total"`      // 总数量
	Page       int                      `json:"page"`       // 当前页码
	PageSize   int                      `json:"pageSize"`   // 每页大小
	TotalPages int                      `json:"totalPages"` // 总页数
	HasNext    bool                     `json:"hasNext"`    // 是否有下一页
}

// GetUserTokensList 获取用户推送令牌列表（支持分页）
func (ps *PebbleService) GetUserTokensList(page, pageSize int) (*PaginatedUserTokens, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100 // 限制最大页面大小
	}

	// 获取用户令牌集合的数据库
	db, err := ps.getCollectionDB(CollectionUserTokens)
	if err != nil {
		return nil, fmt.Errorf("获取用户令牌集合数据库失败: %w", err)
	}

	// 创建迭代器
	iter, err := db.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("创建迭代器失败: %w", err)
	}
	defer iter.Close()

	var allUsers []*models.UserPushTokens

	// 遍历所有用户令牌
	for iter.First(); iter.Valid(); iter.Next() {
		// 解析值
		var userTokens models.UserPushTokens
		if err := json.Unmarshal(iter.Value(), &userTokens); err != nil {
			log.Printf("⚠️ 跳过解析失败的记录: %s, 错误: %v", string(iter.Key()), err)
			continue
		}

		allUsers = append(allUsers, &userTokens)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("迭代器错误: %w", err)
	}

	// 计算分页
	total := len(allUsers)
	totalPages := (total + pageSize - 1) / pageSize
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize

	if startIndex >= total {
		// 页码超出范围，返回空结果
		return &PaginatedUserTokens{
			Users:      []*models.UserPushTokens{},
			Total:      total,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: totalPages,
			HasNext:    false,
		}, nil
	}

	if endIndex > total {
		endIndex = total
	}

	// 获取当前页的数据
	pageUsers := allUsers[startIndex:endIndex]
	hasNext := page < totalPages

	log.Printf("📖 已获取用户令牌列表: 第%d页/%d页, 每页%d条, 当前页%d条, 总共%d条",
		page, totalPages, pageSize, len(pageUsers), total)

	return &PaginatedUserTokens{
		Users:      pageUsers,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasNext:    hasNext,
	}, nil
}

// GetUserTokensListGlobal 全局方法：获取用户推送令牌列表（支持分页）
func GetUserTokensListGlobal(page, pageSize int) (*PaginatedUserTokens, error) {
	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.GetUserTokensList(page, pageSize)
}

// CollectionInfo 集合信息
type CollectionInfo struct {
	Name  string `json:"name"`  // 集合名称
	Count int    `json:"count"` // 记录数量
}

// ListCollections 列出所有集合及其记录数量
func (ps *PebbleService) ListCollections() ([]*CollectionInfo, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if ps.collectionMgr == nil {
		return nil, fmt.Errorf("集合管理器未初始化")
	}

	collectionNames := []string{
		CollectionUserTokens,
		CollectionDevices,
		CollectionBlockedChats,
		CollectionNotifiedPins,
	}

	var result []*CollectionInfo
	for _, name := range collectionNames {
		count, err := ps.getCollectionCount(name)
		if err != nil {
			log.Printf("⚠️ 获取集合 %s 记录数失败: %v", name, err)
			count = 0
		}
		result = append(result, &CollectionInfo{
			Name:  name,
			Count: count,
		})
	}

	log.Printf("📊 集合统计: %+v", result)
	return result, nil
}

// getCollectionCount 获取指定集合的记录数量
func (ps *PebbleService) getCollectionCount(collectionName string) (int, error) {
	db, err := ps.getCollectionDB(collectionName)
	if err != nil {
		return 0, err
	}

	iter, err := db.NewIter(nil)
	if err != nil {
		return 0, err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}

	return count, iter.Error()
}

// ClearCollection 清空指定集合的所有数据
func (ps *PebbleService) ClearCollection(collectionName string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if collectionName == "" {
		return fmt.Errorf("集合名称不能为空")
	}

	// 获取指定集合的数据库
	db, err := ps.getCollectionDB(collectionName)
	if err != nil {
		return fmt.Errorf("获取集合数据库失败: %w", err)
	}

	// 创建迭代器
	iter, err := db.NewIter(nil)
	if err != nil {
		return fmt.Errorf("创建迭代器失败: %w", err)
	}
	defer iter.Close()

	// 收集要删除的键
	var keysToDelete [][]byte
	for iter.First(); iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keysToDelete = append(keysToDelete, key)
	}

	if err := iter.Error(); err != nil {
		return fmt.Errorf("迭代器错误: %w", err)
	}

	// 批量删除
	batch := db.NewBatch()
	for _, key := range keysToDelete {
		if err := batch.Delete(key, nil); err != nil {
			batch.Close()
			return fmt.Errorf("添加删除操作到批处理失败: %w", err)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		batch.Close()
		return fmt.Errorf("提交批处理删除失败: %w", err)
	}

	batch.Close()
	log.Printf("🗑️ 已清空集合 %s，删除了 %d 条记录", collectionName, len(keysToDelete))
	return nil
}

// GetCollectionSize 获取指定集合的记录数量
func (ps *PebbleService) GetCollectionSize(collectionName string) (int, error) {
	return ps.getCollectionCount(collectionName)
}

// ListCollectionsGlobal 全局方法：列出所有集合及其记录数量
func ListCollectionsGlobal() ([]*CollectionInfo, error) {
	service := GetGlobalService()
	if service == nil {
		return nil, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return nil, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.ListCollections()
}

// ClearCollectionGlobal 全局方法：清空指定集合的所有数据
func ClearCollectionGlobal(collectionName string) error {
	service := GetGlobalService()
	if service == nil {
		return fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.ClearCollection(collectionName)
}

// GetCollectionSizeGlobal 全局方法：获取指定集合的记录数量
func GetCollectionSizeGlobal(collectionName string) (int, error) {
	service := GetGlobalService()
	if service == nil {
		return 0, fmt.Errorf("全局 Pebble 服务未初始化，请先初始化推送中心")
	}
	if !service.IsInitialized() {
		return 0, fmt.Errorf("Pebble 服务未正确初始化")
	}
	return service.GetCollectionSize(collectionName)
}

// ===== 屏蔽聊天相关方法 =====

// AddBlockedChat 添加屏蔽聊天
func (ps *PebbleService) AddBlockedChat(userId, chatId, chatType, reason string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if userId == "" || chatId == "" {
		return fmt.Errorf("UserID 和 ChatID 不能为空")
	}

	// 获取屏蔽聊天集合的数据库
	db, err := ps.getCollectionDB(CollectionBlockedChats)
	if err != nil {
		return fmt.Errorf("获取屏蔽聊天集合数据库失败: %w", err)
	}

	// 获取用户现有的屏蔽列表
	userBlockedChats, err := ps.getUserBlockedChatsFromDB(db, userId)
	if err != nil {
		return fmt.Errorf("获取用户屏蔽列表失败: %w", err)
	}

	// 检查是否已经屏蔽过该聊天
	for _, blockedChat := range userBlockedChats.BlockedChats {
		if blockedChat.ChatID == chatId {
			log.Printf("⚠️ 用户 %s 已经屏蔽了聊天 %s", userId, chatId)
			return nil // 已经屏蔽，直接返回成功
		}
	}

	// 添加新的屏蔽聊天
	newBlockedChat := models.BlockedChat{
		UserID:    userId,
		ChatID:    chatId,
		ChatType:  chatType,
		BlockedAt: time.Now().Unix(),
		Reason:    reason,
	}

	userBlockedChats.BlockedChats = append(userBlockedChats.BlockedChats, newBlockedChat)
	userBlockedChats.UpdatedAt = time.Now().Unix()

	// 序列化为 JSON
	data, err := json.Marshal(userBlockedChats)
	if err != nil {
		return fmt.Errorf("序列化用户屏蔽列表失败: %w", err)
	}

	// 保存到数据库
	key := getUserBlockedChatsKey(userId)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return fmt.Errorf("保存用户屏蔽列表失败: %w", err)
	}

	log.Printf("✅ 已添加屏蔽聊天: UserID=%s, ChatID=%s, ChatType=%s", userId, chatId, chatType)
	return nil
}

// IsBlockedChat 检查聊天是否被屏蔽
func (ps *PebbleService) IsBlockedChat(userId, chatId string) (bool, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if userId == "" || chatId == "" {
		return false, fmt.Errorf("UserID 和 ChatID 不能为空")
	}

	// 获取屏蔽聊天集合的数据库
	db, err := ps.getCollectionDB(CollectionBlockedChats)
	if err != nil {
		return false, fmt.Errorf("获取屏蔽聊天集合数据库失败: %w", err)
	}

	// 获取用户屏蔽列表
	userBlockedChats, err := ps.getUserBlockedChatsFromDB(db, userId)
	if err != nil {
		return false, fmt.Errorf("获取用户屏蔽列表失败: %w", err)
	}

	// 检查是否屏蔽了该聊天
	for _, blockedChat := range userBlockedChats.BlockedChats {
		if blockedChat.ChatID == chatId {
			return true, nil // 已屏蔽
		}
	}

	return false, nil // 未屏蔽
}

// RemoveBlockedChat 移除屏蔽聊天
func (ps *PebbleService) RemoveBlockedChat(userId, chatId string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if userId == "" || chatId == "" {
		return fmt.Errorf("UserID 和 ChatID 不能为空")
	}

	// 获取屏蔽聊天集合的数据库
	db, err := ps.getCollectionDB(CollectionBlockedChats)
	if err != nil {
		return fmt.Errorf("获取屏蔽聊天集合数据库失败: %w", err)
	}

	// 获取用户现有的屏蔽列表
	userBlockedChats, err := ps.getUserBlockedChatsFromDB(db, userId)
	if err != nil {
		return fmt.Errorf("获取用户屏蔽列表失败: %w", err)
	}

	// 查找并移除指定的屏蔽聊天
	found := false
	var newBlockedChats []models.BlockedChat
	for _, blockedChat := range userBlockedChats.BlockedChats {
		if blockedChat.ChatID != chatId {
			newBlockedChats = append(newBlockedChats, blockedChat)
		} else {
			found = true
		}
	}

	if !found {
		log.Printf("⚠️ 用户 %s 没有屏蔽聊天 %s", userId, chatId)
		return nil // 没有屏蔽，直接返回成功
	}

	// 更新屏蔽列表
	userBlockedChats.BlockedChats = newBlockedChats
	userBlockedChats.UpdatedAt = time.Now().Unix()

	// 如果列表为空，删除整个记录
	if len(userBlockedChats.BlockedChats) == 0 {
		key := getUserBlockedChatsKey(userId)
		if err := db.Delete(key, pebble.Sync); err != nil {
			return fmt.Errorf("删除用户屏蔽列表失败: %w", err)
		}
	} else {
		// 序列化为 JSON 并保存
		data, err := json.Marshal(userBlockedChats)
		if err != nil {
			return fmt.Errorf("序列化用户屏蔽列表失败: %w", err)
		}

		key := getUserBlockedChatsKey(userId)
		if err := db.Set(key, data, pebble.Sync); err != nil {
			return fmt.Errorf("保存用户屏蔽列表失败: %w", err)
		}
	}

	log.Printf("✅ 已移除屏蔽聊天: UserID=%s, ChatID=%s", userId, chatId)
	return nil
}

// GetUserBlockedChats 获取用户的所有屏蔽聊天
func (ps *PebbleService) GetUserBlockedChats(userId string) (*models.UserBlockedChats, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if userId == "" {
		return nil, fmt.Errorf("UserID 不能为空")
	}

	// 获取屏蔽聊天集合的数据库
	db, err := ps.getCollectionDB(CollectionBlockedChats)
	if err != nil {
		return nil, fmt.Errorf("获取屏蔽聊天集合数据库失败: %w", err)
	}

	// 获取用户屏蔽列表
	userBlockedChats, err := ps.getUserBlockedChatsFromDB(db, userId)
	if err != nil {
		return nil, fmt.Errorf("获取用户屏蔽列表失败: %w", err)
	}

	log.Printf("📖 已获取用户屏蔽聊天列表: UserID=%s, 数量=%d", userId, len(userBlockedChats.BlockedChats))
	return userBlockedChats, nil
}

// ===== PIN通知相关方法 =====

// AddNotifiedPin 添加已通知的PIN
func (ps *PebbleService) AddNotifiedPin(pinId string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if pinId == "" {
		return fmt.Errorf("PinID 不能为空")
	}

	// 获取已通知PIN集合的数据库
	db, err := ps.getCollectionDB(CollectionNotifiedPins)
	if err != nil {
		return fmt.Errorf("获取已通知PIN集合数据库失败: %w", err)
	}

	// 创建已通知PIN信息
	notifiedPin := &models.NotifiedPin{
		PinID: pinId,
		// ChatID:      chatId,
		// UserID:      userId,
		NotifiedAt: time.Now().Unix(),
		// MessageHash: messageHash,
	}

	// 序列化为 JSON
	data, err := json.Marshal(notifiedPin)
	if err != nil {
		return fmt.Errorf("序列化已通知PIN信息失败: %w", err)
	}

	// 保存到数据库
	key := getNotifiedPinKey(pinId)
	if err := db.Set(key, data, pebble.Sync); err != nil {
		return fmt.Errorf("保存已通知PIN信息失败: %w", err)
	}

	log.Printf("✅ 已添加已通知PIN: PinID=%s", pinId)
	return nil
}

// IsNotifiedPin 检查PIN是否已通知
func (ps *PebbleService) IsNotifiedPin(pinId string) (bool, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if pinId == "" {
		return false, fmt.Errorf("PinID 不能为空")
	}

	// 获取已通知PIN集合的数据库
	db, err := ps.getCollectionDB(CollectionNotifiedPins)
	if err != nil {
		return false, fmt.Errorf("获取已通知PIN集合数据库失败: %w", err)
	}

	key := getNotifiedPinKey(pinId)
	_, closer, err := db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return false, nil // 未通知
		}
		return false, fmt.Errorf("检查PIN通知状态失败: %w", err)
	}
	closer.Close()

	return true, nil // 已通知
}

// RemoveNotifiedPin 移除已通知PIN记录
func (ps *PebbleService) RemoveNotifiedPin(pinId string) error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if pinId == "" {
		return fmt.Errorf("PinID 不能为空")
	}

	// 获取已通知PIN集合的数据库
	db, err := ps.getCollectionDB(CollectionNotifiedPins)
	if err != nil {
		return fmt.Errorf("获取已通知PIN集合数据库失败: %w", err)
	}

	key := getNotifiedPinKey(pinId)
	if err := db.Delete(key, pebble.Sync); err != nil {
		return fmt.Errorf("移除已通知PIN失败: %w", err)
	}

	log.Printf("✅ 已移除已通知PIN: PinID=%s", pinId)
	return nil
}

// PebbleTokenStore 基于 Pebble 的用户令牌存储实现
type PebbleTokenStore struct {
	service *PebbleService
}

// NewPebbleTokenStore 创建基于 Pebble 的令牌存储
func NewPebbleTokenStore(service *PebbleService) *PebbleTokenStore {
	return &PebbleTokenStore{
		service: service,
	}
}

// NewGlobalPebbleTokenStore 创建基于全局 Pebble 服务的令牌存储
func NewGlobalPebbleTokenStore() *PebbleTokenStore {
	service := GetGlobalService()
	if service == nil {
		log.Printf("❌ 全局 Pebble 服务未初始化，无法创建令牌存储")
		return nil
	}
	if !service.IsInitialized() {
		log.Printf("❌ Pebble 服务未正确初始化，无法创建令牌存储")
		return nil
	}
	return &PebbleTokenStore{
		service: service,
	}
}

// convertToServiceUserTokens 将 models.UserPushTokens 转换为 push_service.UserPushTokens
func convertToServiceUserTokens(modelTokens *models.UserPushTokens) *push_service.UserPushTokens {
	return &push_service.UserPushTokens{
		MetaID:    modelTokens.MetaID,
		Tokens:    modelTokens.Tokens,
		UpdatedAt: time.Unix(modelTokens.UpdatedAt, 0),
	}
}

// convertFromServiceUserTokens 将 push_service.UserPushTokens 转换为 models.UserPushTokens
func convertFromServiceUserTokens(serviceTokens *push_service.UserPushTokens) *models.UserPushTokens {
	return &models.UserPushTokens{
		MetaID:    serviceTokens.MetaID,
		Tokens:    serviceTokens.Tokens,
		UpdatedAt: serviceTokens.UpdatedAt.Unix(),
	}
}

// GetUserTokens 根据metaId获取用户的所有推送令牌 (实现 UserTokenStore 接口)
func (pts *PebbleTokenStore) GetUserTokens(ctx context.Context, metaId string) (*push_service.UserPushTokens, error) {
	modelTokens, err := pts.service.GetUserTokens(metaId)
	if err != nil {
		return nil, err
	}
	return convertToServiceUserTokens(modelTokens), nil
}

// SetUserToken 设置用户在指定平台的推送令牌 (实现 UserTokenStore 接口)
func (pts *PebbleTokenStore) SetUserToken(ctx context.Context, metaId string, platform string, token string) error {
	// 直接调用，因为现在 token 本身就是设备ID
	return pts.service.SetUserToken(metaId, platform, token)
}

// RemoveUserToken 移除用户在指定平台的推送令牌 (实现 UserTokenStore 接口)
func (pts *PebbleTokenStore) RemoveUserToken(ctx context.Context, metaId string, platform string) error {
	return pts.service.RemoveUserToken(metaId, platform)
}

// GetAllUserTokens 获取所有用户的令牌 (实现 UserTokenStore 接口)
func (pts *PebbleTokenStore) GetAllUserTokens(ctx context.Context, metaIds []string) (map[string]*push_service.UserPushTokens, error) {
	modelTokensMap, err := pts.service.GetAllUserTokens(metaIds)
	if err != nil {
		return nil, err
	}

	result := make(map[string]*push_service.UserPushTokens)
	for metaId, modelTokens := range modelTokensMap {
		result[metaId] = convertToServiceUserTokens(modelTokens)
	}

	return result, nil
}
