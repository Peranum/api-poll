# Upbit API Poll Service

A high-performance Go service that continuously monitors Upbit cryptocurrency exchange announcements and detects new market listings in real-time. The service is designed to quickly capture new listing announcements for trading automation purposes.

## Overview

The Upbit API Poll service polls the Upbit announcements API at configurable intervals using a pool of rotating proxy servers. When new announcements are detected, particularly "Market Support" announcements indicating new cryptocurrency listings, the service immediately notifies connected systems and can trigger automated trading actions.

### Key Features

- **Real-time announcement monitoring** with configurable polling rates
- **Proxy rotation** for distributed load and rate limiting compliance
- **WebSocket API** for real-time news streaming to clients
- **Telegram notifications** for important announcements
- **gRPC integration** with gate-exchange service for automated trading
- **Work schedule support** with timezone-aware operation hours
- **Comprehensive metrics and logging**

## Core Components

### APIPoller

The `APIPoller` is the heart of the service responsible for:

- **High-frequency polling** of Upbit's announcements API (`/api/v1/announcements`)
- **Proxy management** with automatic rotation across multiple proxy servers
- **Rate limiting** with configurable RPS (requests per second) targets
- **Retry logic** with exponential backoff for failed requests
- **News detection** by comparing latest announcement titles
- **Performance monitoring** with detailed metrics logging

Key features:

- Supports multiple HTTP proxies for distributed requests
- Configurable target RPS with automatic proxy allocation
- Thread-safe news detection with mutex locks
- Request buffering and connection pooling for optimal performance

### NewsMonitor

The `NewsMonitor` component handles:

- **Real-time news processing** from the APIPoller stream
- **Pattern matching** for specific announcement types (e.g., "Market Support for...")
- **Ticker extraction** from announcement titles
- **Automated trading integration** via gRPC calls to gate-exchange service
- **Background monitoring** with graceful shutdown handling

The monitor specifically watches for "Market Support" announcements which indicate new cryptocurrency listings on Upbit, triggering immediate trading actions.

## Configuration

### Configuration Structure

The service uses a hierarchical configuration system with YAML files and environment variable overrides.

#### Adding New Configuration Variables

1. **Add to config struct** with appropriate tags:

```go
type App struct {
    NewField string `mapstructure:"new_field" validate:"required" env:"NEW_FIELD"`
}
```

2. **Update YAML configuration** in `configs/local.yaml` and `configs/remote.yaml`:

```yaml
app:
  new_field: "default_value"
```

3. **Add environment variable** to `.env.example`:

```bash
UPBITAP_NEW_FIELD=example_value
```

### Environment Variables

Configuration values are first loaded from YAML files based on `mapstructure` tags, then can be overridden by environment variables.

**Important**: Environment variables must be prefixed with `UPBITAP_` while the `env` tags in structs don't include this prefix.

Example:

- Struct tag: `env:"SERVE_ADDRESS"`
- Environment variable: `UPBITAP_SERVE_ADDRESS`

Create a `.env.example` file with with examples of environment variables:

```bash
# App Configuration
UPBITAP_APP_SERVE_ADDRESS=localhost:8080
UPBITAP_APP_TARGET_RPS=5.0
UPBITAP_APP_METRICS_CHECK_PERIOD=5m
UPBITAP_APP_UPBIT_ANNOUNCEMENTS_URL=https://api-manager.upbit.com/api/v1/announcements?os=web&page=1&per_page=1&category=екфву
UPBITAP_APP_SINGLE_PROXY_MAX_RPS=0.2
UPBITAP_APP_API_POLLER_MAX_RETRIES=3
UPBITAP_APP_API_POLLER_RETRY_DELAY=75ms
UPBITAP_APP_API_POLLER_RETRY_DELAY_MULTIPLIER=1.0

# Telegram Configuration
UPBITAP_TELEGRAM_BOT_ID=your_bot_id
UPBITAP_TELEGRAM_AUTHORIZATION_TOKEN=your_bot_token
UPBITAP_TELEGRAM_GROUP_ID=-1234567890

# Logger Configuration
UPBITAP_LOGGER_LEVEL=info
UPBITAP_LOGGER_FORMAT=json
UPBITAP_LOGGER_ADD_SOURCE=false

# gRPC Configuration
UPBITAP_GRPC_ADDRESS=gate-exchange:49999
UPBITAP_GRPC_DIAL_TIMEOUT=5s
UPBITAP_GRPC_CALL_TIMEOUT=10s
```

## Installation & Usage

### Prerequisites

- Go 1.24.3+
- Docker (optional)
- Telegram Bot Token (for notifications)
- HTTP Proxy servers (for Upbit API access)

### Local Development

1. **Clone the repository**:

```bash
git clone <repository-url>
cd upbit-api-poll
```

2. **Install dependencies**:

```bash
go mod tidy
```

3. **Configure the service**:

```bash
cp configs/local.yaml configs/local.yaml.bak
# Edit configs/local.yaml with your settings
```

4. **Set environment variables**:

```bash
cp .env.example .env
# Edit .env with your configuration
```

5. **Build and run**:

```bash
make build
./bin/upbit-api-poll --config configs/local.yaml
```

### Docker Deployment

1. **Build Docker image**:

```bash
docker build -t upbit-api-poll .
```

2. **Run with Docker Compose**:

```bash
docker-compose up -d upbit-api-poll
```

The service will be available on port 8080 with WebSocket endpoint at `/ws/news`.

### Available Commands

```bash
# Format code
make fmt

# Generate protobuf code
make proto

# Build binary
make build

# Run tests
make test
```

## API Endpoints

### WebSocket API

- **`/ws/news`** - Real-time news stream
  - Sends JSON messages with new announcements
  - Includes keep-alive messages every 30 seconds
  - Message format:
    ```json
    {
      "is_keep_alive": false,
      "news": "Market Support for Livepeer(LPT)(KRW, USDT Market)"
    }
    ```

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   APIPoller     │───▶│   NewsMonitor   │───▶│  Gate Exchange  │
│                 │    │                 │    │    (gRPC)       │
│ • Proxy Pool    │    │ • Pattern Match │    │ • Auto Trading  │
│ • Rate Limiting │    │ • Ticker Extract│    │                 │
│ • News Detection│    │ • Trade Trigger │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │
         ▼                       ▼
┌─────────────────┐    ┌─────────────────┐
│  WebSocket API  │    │    Telegram     │
│                 │    │  Notifications  │
│ • Real-time     │    │                 │
│ • Client Stream │    │ • Alert System  │
└─────────────────┘    └─────────────────┘
```

## Monitoring & Metrics

The service provides comprehensive monitoring:

- **RPS Metrics**: Real-time requests per second tracking
- **Active Connections**: Concurrent goroutine monitoring
- **Proxy Performance**: Individual proxy success rates
- **Response Times**: Detailed operation timing logs
- **Error Tracking**: Failed request and retry statistics

Metrics are logged every 5 minutes (configurable) and include:

- Total request count
- Actual vs target RPS
- Active goroutines
- Proxy pool utilization

## Proxy Configuration

The service requires HTTP proxies for accessing Upbit's API. Configure proxies in the YAML file:

```yaml
app:
  proxies:
    - username: "proxy_user"
      password: "proxy_pass"
      host: "proxy.example.com"
      port: 8080
```

**Important**: Ensure you have enough proxies to meet your target RPS:

- Required proxies = `target_rps / single_proxy_max_rps`
- Each proxy is limited to `single_proxy_max_rps` (default: 0.2 RPS)

## Work Schedule

The service supports timezone-aware work schedules to operate only during specified hours:

```yaml
app:
  work_schedule:
    time_zone: "Asia/Seoul"
    schedule:
      monday:
        start_time: "08:00"
        end_time: "23:59"
        preparation_time: "5m"
```

During non-work hours, the service will sleep and resume automatically.

## Error Handling

The service implements robust error handling:

- **Request retries** with exponential backoff
- **Proxy failover** for connection issues
- **Graceful degradation** during API outages
- **Telegram alerts** for critical errors
- **Automatic recovery** after network issues

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make fmt`
6. Submit a pull request

## Support

For issues and questions:

- Create an issue in the repository
- Contact the development team via Telegram
- Check logs for detailed error information
