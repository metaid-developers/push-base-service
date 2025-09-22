package conf

import (
	"fmt"

	"github.com/spf13/viper"
)

var (
	Net  string = ""
	Port string = ""

	RdsDsn          string = ""
	RdsMaxOpenConns int    = 0
	RdsMaxIgleConns int    = 0

	// API Key for authentication
	APIKey = ""

	// Push Center Configuration
	PushCenterEnabled bool   = false
	PushCenterDBPath  string = ""

	// Socket Client Configuration
	SocketServerURL        string = ""
	SocketExtraPushAuthKey string = ""
	SocketPath             string = ""
	SocketTimeout          int    = 0

	// Push Service Configuration
	PushDefaultProvider     string = ""
	PushDefaultSound        string = ""
	PushDefaultTTL          int    = 0
	PushDefaultPriority     string = ""
	PushMaxRetries          int    = 0
	PushRetryInterval       string = ""
	PushBatchSize           int    = 0
	PushBatchTimeout        string = ""
	PushMaxConcurrency      int    = 0
	PushEnableStats         bool   = false
	PushStatsInterval       string = ""
	PushHealthCheckInterval string = ""

	// Expo Provider Configuration
	ExpoAccessToken     string = ""
	ExpoTimeout         string = ""
	ExpoMaxRetries      int    = 0
	ExpoBaseDelay       string = ""
	ExpoDefaultSound    string = ""
	ExpoDefaultTTL      int    = 0
	ExpoDefaultPriority string = ""
	ExpoBatchSize       int    = 0
	ExpoMaxConcurrency  int    = 0
)

func InitConfig(configPath string) {
	if configPath == "" {
		configPath = GetYaml()
	}
	// Set the file name of the configurations file
	fmt.Printf("configPath:%s\n", configPath)
	viper.SetConfigFile(configPath)
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	Net = viper.GetString("net")
	Port = viper.GetString("port")

	RdsDsn = viper.GetString("rds.dsn")
	RdsMaxOpenConns = viper.GetInt("rds.max_open_conns")
	RdsMaxIgleConns = viper.GetInt("rds.max_igle_conns")

	// 读取 API Key 配置
	APIKey = viper.GetString("api_key")

	// 读取推送中心配置
	PushCenterEnabled = viper.GetBool("push_center.enabled")
	PushCenterDBPath = viper.GetString("push_center.db_path")

	// 读取 Socket 客户端配置
	SocketServerURL = viper.GetString("socket_client.server_url")
	SocketExtraPushAuthKey = viper.GetString("socket_client.extra_push_auth_key")
	SocketPath = viper.GetString("socket_client.path")
	SocketTimeout = viper.GetInt("socket_client.timeout")

	// 读取推送服务配置
	PushDefaultProvider = viper.GetString("push.default_provider")
	PushDefaultSound = viper.GetString("push.default_sound")
	PushDefaultTTL = viper.GetInt("push.default_ttl")
	PushDefaultPriority = viper.GetString("push.default_priority")
	PushMaxRetries = viper.GetInt("push.max_retries")
	PushRetryInterval = viper.GetString("push.retry_interval")
	PushBatchSize = viper.GetInt("push.batch_size")
	PushBatchTimeout = viper.GetString("push.batch_timeout")
	PushMaxConcurrency = viper.GetInt("push.max_concurrency")
	PushEnableStats = viper.GetBool("push.enable_stats")
	PushStatsInterval = viper.GetString("push.stats_interval")
	PushHealthCheckInterval = viper.GetString("push.health_check_interval")

	// 读取 Expo 提供者配置
	ExpoAccessToken = viper.GetString("push.providers.expo.access_token")
	ExpoTimeout = viper.GetString("push.providers.expo.timeout")
	ExpoMaxRetries = viper.GetInt("push.providers.expo.max_retries")
	ExpoBaseDelay = viper.GetString("push.providers.expo.base_delay")
	ExpoDefaultSound = viper.GetString("push.providers.expo.default_sound")
	ExpoDefaultTTL = viper.GetInt("push.providers.expo.default_ttl")
	ExpoDefaultPriority = viper.GetString("push.providers.expo.default_priority")
	ExpoBatchSize = viper.GetInt("push.providers.expo.batch_size")
	ExpoMaxConcurrency = viper.GetInt("push.providers.expo.max_concurrency")
}
