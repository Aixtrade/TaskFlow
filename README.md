# TaskFlow

[中文](README_CN.md)

A distributed task queue system built with Go, using Redis and Asynq for reliable background job processing.

## Features

- **RESTful API** - HTTP API for task management (create, query, cancel, delete)
- **Distributed Workers** - Scalable worker pool with configurable concurrency
- **Priority Queues** - Support for critical, high, default, and low priority queues
- **Scheduled Tasks** - Schedule tasks for future execution
- **Task Deduplication** - Unique task constraints to prevent duplicate processing
- **Retry Mechanism** - Automatic retry with configurable max retries
- **Observability** - Structured logging
- **Health Checks** - Health, readiness, and liveness endpoints

## Quick Start

### Prerequisites

- Go 1.25.1
- Redis 6.0+
- Docker (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/Aixtrade/TaskFlow.git
cd TaskFlow

# Install dependencies
make deps

# Build binaries
make build
```

### Running with Docker

```bash
# Start Redis
make redis-up

# Or use Docker Compose for full stack
make docker-up
```

### Running Locally

1. Start Redis:
```bash
make redis-up
```

2. Start the API server:
```bash
make run-api
```

3. Start the Worker server (in another terminal):
```bash
make run-server
```

### Create Your First Task

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "demo",
    "payload": {
      "message": "Hello, TaskFlow!",
      "count": 3
    }
  }'
```

## Project Structure

```
TaskFlow/
├── cmd/
│   ├── api/           # API server entry point
│   └── server/        # Worker server entry point
├── configs/           # Configuration files
├── deployments/       # Docker and deployment files
├── docs/              # Documentation
├── internal/
│   ├── application/   # Application services (CQRS)
│   ├── config/        # Configuration loading
│   ├── domain/        # Domain entities and interfaces
│   ├── infrastructure/# External dependencies (Redis, Asynq)
│   ├── interfaces/    # HTTP handlers and DTOs
│   └── worker/        # Task handlers and registry
└── pkg/
    ├── errors/        # Common error types
    ├── payload/       # Task payload definitions
    └── tasktype/      # Task type constants
```

## Configuration

TaskFlow uses YAML configuration with environment variable overrides. Create a local config from the example:

```bash
cp configs/config.yaml.example configs/config.yaml
```

The local `configs/*.yaml` files are ignored by git.

Example structure:

```yaml
app:
  name: taskflow
  env: production

server:
  http:
    host: 0.0.0.0
    port: 8080
  worker:
    concurrency: 10
    health:
      enabled: true
      host: 0.0.0.0
      port: 8082

redis:
  addr: localhost:6379
  password: ""
  db: 0

queues:
  critical: 10
  high: 5
  default: 3
  low: 1

progress:
  max_len: 1000
  ttl: 1h
  read_timeout: 30s
```

Environment variables use the `TASKFLOW_` prefix:
- `TASKFLOW_REDIS_ADDR`
- `TASKFLOW_SERVER_HTTP_PORT`
- etc.

## Documentation

- [Architecture](docs/architecture.md) - System design and components
- [API Reference](docs/api.md) - REST API documentation
- [Creating Tasks](docs/creating-tasks.md) - Guide to creating custom tasks

## Development

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage

# Run linter
make lint

# Clean build artifacts
make clean
```

## Monitoring

- **Health Check**: `GET /health`
- **Readiness Check**: `GET /ready`
- **Liveness Check**: `GET /live`
- **Asynqmon UI**: `make asynqmon` to start the web dashboard

## License

MIT License
