# Push Base Service

## Overview

Push Base Service is a notification push service designed to provide app push notifications for MetaSO's chat nodes. This service supports real-time message delivery for private chat and group chat scenarios, enabling seamless communication experiences in mobile applications.

## Features

- **Multi-Platform Push Support**: Supports Expo Push Notification service
- **Real-time Message Delivery**: Integrates with Socket.IO for real-time message reception
- **Chat Message Push**: Supports both private chat and group chat message notifications
- **Token Management**: Efficient user push token storage and management using Pebble database
- **RESTful API**: Provides HTTP API for push operations and token management
- **Swagger Documentation**: Built-in API documentation
- **High Performance**: Concurrent push delivery with configurable batch processing
- **Retry Mechanism**: Automatic retry with configurable retry intervals
- **Environment Support**: Supports mainnet and testnet environments

## Technology Stack

- **Language**: Go 1.24.7
- **Web Framework**: Gin
- **Database**: Pebble (for token storage)
- **Push Service**: Expo Push Notification
- **Real-time Communication**: Socket.IO
- **Configuration**: Viper
- **API Documentation**: Swagger

## Quick Start

### Prerequisites

- Go 1.24.7 or higher
- Access to MetaSO chat node Socket.IO server
- Expo access token (for Expo push notifications)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd push-base-service
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the service:
   - Copy `conf/conf_example.yaml` to `conf/conf_pro.yaml` or `conf/conf_mainnet.yaml`
   - Update the configuration with your settings:
     - Socket.IO server URL
     - Expo access token
     - Database path
     - API port

4. Run the service:
```bash
# Run with mainnet environment
go run main.go -env mainnet

# Run with testnet environment
go run main.go -env testnet
```

### Docker

You can also run the service using Docker:

```bash
# Build for mainnet
docker build -f deploy/DockerfileMainnet -t push-base-service:mainnet .

# Build for testnet
docker build -f deploy/DockerfileTestnet -t push-base-service:testnet .

# Run
docker run -p 1234:1234 push-base-service:mainnet
```

## Configuration

Key configuration items in `conf_example.yaml`:

```yaml
# API port
port: "1234"

# Push service configuration
push:
  default_provider: "expo"
  providers:
    expo:
      access_token: "your-expo-access-token"
      timeout: "30s"
      max_retries: 3

# Push center configuration
push_center:
  enabled: true
  db_path: "./data/push_center_pebble"

# Socket.IO client configuration
socket_client:
  server_url: "https://your-socket-server-url"
  extra_push_auth_key: "your-auth-key"
  path: "/socket/socket.io/"
  timeout: 10
```

## API Documentation

Once the service is running, you can access the Swagger API documentation at:

```
http://localhost:1234/swagger/index.html
```

## Architecture

The service consists of several key components:

- **Push Center**: Core service that manages push notifications
- **Socket Client Service**: Connects to MetaSO chat node via Socket.IO
- **Push Service**: Handles push notification delivery
- **Expo Service**: Expo push notification provider implementation
- **Pebble Service**: Database service for token storage
- **HTTP Controller**: RESTful API endpoints

## License

See [LICENSE](LICENSE) file for details.
