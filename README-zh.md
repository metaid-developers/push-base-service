# 推送基础服务

[English](README.md) | [中文](README-zh.md)

## 项目简介

推送基础服务（Push Base Service）是一个专为 MetaSO 的 chat 节点提供 app 推送通知的服务。该服务支持私聊和群聊场景的实时消息推送，为移动应用提供无缝的通信体验。

## 功能特性

- **多平台推送支持**: 支持 Expo 推送通知服务
- **实时消息推送**: 通过 Socket.IO 集成实现实时消息接收
- **聊天消息推送**: 支持私聊和群聊消息通知
- **令牌管理**: 使用 Pebble 数据库高效存储和管理用户推送令牌
- **RESTful API**: 提供用于推送操作和令牌管理的 HTTP API
- **Swagger 文档**: 内置 API 文档
- **高性能**: 支持并发推送，可配置批量处理
- **重试机制**: 自动重试，可配置重试间隔
- **环境支持**: 支持主网和测试网环境


## 快速开始

### 前置要求

- Go 1.24.7 或更高版本
- 访问 MetaSO chat 节点的 Socket.IO 服务器
- Expo access token (用于 Expo 推送通知)

### 安装步骤

1. 克隆仓库:
```bash
git clone <repository-url>
cd push-base-service
```

2. 安装依赖:
```bash
go mod download
```

3. 配置服务:
   - 复制 `conf/conf_example.yaml` 到 `conf/conf_pro.yaml` 或 `conf/conf_mainnet.yaml`
   - 更新配置信息:
     - Socket.IO 服务器 URL
     - Expo access token
     - 数据库路径
     - API 端口

4. 运行服务:
```bash
# 使用主网环境运行
go run main.go -env mainnet

# 使用测试网环境运行
go run main.go -env testnet
```

### Docker 部署

您也可以使用 Docker 运行服务:

```bash
# 构建主网版本
docker build -f deploy/DockerfileMainnet -t push-base-service:mainnet .

# 构建测试网版本
docker build -f deploy/DockerfileTestnet -t push-base-service:testnet .

# 运行
docker run -p 1234:1234 push-base-service:mainnet
```

## 配置说明

`conf_example.yaml` 中的主要配置项:

```yaml
# API 端口
port: "1234"

# 推送服务配置
push:
  default_provider: "expo"
  providers:
    expo:
      access_token: "your-expo-access-token"
      timeout: "30s"
      max_retries: 3

# 推送中心配置
push_center:
  enabled: true
  db_path: "./data/push_center_pebble"

# Socket.IO 客户端配置
socket_client:
  server_url: "https://your-socket-server-url"
  extra_push_auth_key: "your-auth-key"
  path: "/socket/socket.io/"
  timeout: 10
```

## API 文档

服务启动后，您可以访问 Swagger API 文档:

```
http://localhost:1234/swagger/index.html
```

## 架构说明

服务包含以下核心组件:

- **推送中心 (Push Center)**: 管理推送通知的核心服务
- **Socket 客户端服务**: 通过 Socket.IO 连接到 MetaSO chat 节点
- **推送服务**: 处理推送通知的发送
- **Expo 服务**: Expo 推送通知提供者实现
- **Pebble 服务**: 用于令牌存储的数据库服务
- **HTTP 控制器**: RESTful API 端点

## 许可证

详细信息请参阅 [LICENSE](LICENSE) 文件。
