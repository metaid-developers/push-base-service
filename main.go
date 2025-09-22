package main

import (
	"flag"
	"fmt"
	"log"
	"push-base-service/conf"
	"push-base-service/controller"
	"push-base-service/service/expo_service"
	"push-base-service/service/pebble_service"
	pushcenter "push-base-service/service/push_center"
	"push-base-service/service/socket_client_service"
	"time"
)

func initPushCenter() {
	// 检查是否启用推送中心
	if !conf.PushCenterEnabled {
		log.Printf("📴 推送中心未启用，跳过初始化")
		return
	}

	log.Printf("🚀 开始初始化推送中心...")

	// 1. 创建 Socket 客户端配置
	socketConfig := &socket_client_service.Config{
		ServerURL:        conf.SocketServerURL,
		ExtraPushAuthKey: conf.SocketExtraPushAuthKey,
		Path:             conf.SocketPath,
		Timeout:          conf.SocketTimeout,
	}

	// 设置默认值
	if socketConfig.Path == "" {
		socketConfig.Path = "/socket.io/"
	}
	if socketConfig.Timeout == 0 {
		socketConfig.Timeout = 10
	}

	// 2. 创建 Pebble 数据库配置
	pebbleConfig := &pebble_service.Config{
		DBPath: conf.PushCenterDBPath,
	}

	// 设置默认数据库路径
	if pebbleConfig.DBPath == "" {
		pebbleConfig.DBPath = "./data/push_center_pebble"
	}

	// 3. 创建推送中心配置
	pushCenterConfig := &pushcenter.Config{
		SocketConfig: socketConfig,
		PebbleConfig: pebbleConfig,
		EnabledTypes: []string{"private_chat", "group_chat"}, // 启用私聊和群聊消息
	}

	// 4. 创建推送中心实例
	pushCenter := pushcenter.NewPushCenter(pushCenterConfig)

	// 5. 初始化推送中心
	if err := pushCenter.Initialize(); err != nil {
		log.Fatalf("❌ 初始化推送中心失败: %v", err)
	}

	// 6. 创建并注册 Expo 推送提供者
	expoConfig := &expo_service.Config{
		AccessToken:     conf.ExpoAccessToken, // 🔑 添加 Access Token
		Timeout:         parseDuration(conf.ExpoTimeout, 30*time.Second),
		MaxRetries:      getIntWithDefault(conf.ExpoMaxRetries, 3),
		BaseDelay:       parseDuration(conf.ExpoBaseDelay, 1*time.Second),
		DefaultSound:    getStringWithDefault(conf.ExpoDefaultSound, "default"),
		DefaultTTL:      getIntWithDefault(conf.ExpoDefaultTTL, 3600),
		DefaultPriority: getStringWithDefault(conf.ExpoDefaultPriority, "normal"),
		BatchSize:       getIntWithDefault(conf.ExpoBatchSize, 100),
		MaxConcurrency:  getIntWithDefault(conf.ExpoMaxConcurrency, 6),
	}

	if err := pushCenter.GetPushManager().RegisterExpoProvider(expoConfig); err != nil {
		log.Printf("⚠️ 注册 Expo 推送提供者失败: %v", err)
	} else {
		log.Printf("✅ 已注册 Expo 推送提供者")
	}

	// 7. 启动推送中心
	go func() {
		if err := pushCenter.Run(); err != nil {
			log.Fatalf("❌ 启动推送中心失败: %v", err)
		}
	}()

	// 8. 等待推送中心启动
	time.Sleep(2 * time.Second)

	if pushCenter.IsRunning() {
		log.Printf("✅ 推送中心已成功启动")
		log.Printf("🔗 Socket 服务器: %s", conf.SocketServerURL)
		log.Printf("🗄️ 数据库路径: %s", conf.PushCenterDBPath)
		log.Printf("🔑 SocketExtraPushAuthKey: %s", conf.SocketExtraPushAuthKey)
	} else {
		log.Printf("⚠️ 推送中心启动状态检查失败")
	}

	// 注册优雅关闭处理
	// 注意：这里只是示例，实际项目中可能需要更完善的信号处理
	log.Printf("💡 提示：推送中心将在应用程序退出时自动关闭")
}

// 辅助函数：解析时间间隔字符串
func parseDuration(durationStr string, defaultDuration time.Duration) time.Duration {
	if durationStr == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("⚠️ 解析时间间隔失败 '%s'，使用默认值: %v", durationStr, defaultDuration)
		return defaultDuration
	}
	return duration
}

// 辅助函数：获取字符串配置值，提供默认值
func getStringWithDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// 辅助函数：获取整数配置值，提供默认值
func getIntWithDefault(value, defaultValue int) int {
	if value == 0 {
		return defaultValue
	}
	return value
}

// Package main
// @title 推送基础服务 API
// @version 1.0
// @description 推送通知基础服务，支持多平台推送和用户令牌管理
// @host api.idchat.io
// @BasePath /push-base
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-KEY
func main() {
	var env string
	flag.StringVar(&env, "env", "mainnet", "env config: testnet, mainnet")
	flag.Parse()

	switch env {
	case "mainnet":
		conf.SystemEnvironmentEnum = conf.MainnetEnvironmentEnum
	case "testnet":
		conf.SystemEnvironmentEnum = conf.TestnetEnvironmentEnum
	default:
		conf.SystemEnvironmentEnum = conf.ExampleEnvironmentEnum
	}

	conf.InitConfig("")

	fmt.Printf("run push-base-service service, env: %s\n", env)

	initPushCenter()

	controller.Run()
}
