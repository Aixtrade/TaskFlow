# Architecture

## System Overview

TaskFlow is a distributed task queue system consisting of two main services:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  API Server │────▶│    Redis    │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
                                        ┌─────────────┐
                                        │   Worker    │
                                        │   Server    │
                                        └─────────────┘
```

## Core Components

### API Server (`cmd/api`)

The API server provides RESTful endpoints for:
- Creating new tasks
- Querying task status
- Cancelling/deleting tasks
- Retrieving queue statistics

Built with:
- **Gin** - HTTP web framework
- **Asynq Client** - Task enqueue client

### Worker Server (`cmd/server`)

The worker server processes tasks from Redis queues:
- Registers task handlers
- Processes tasks concurrently
- Handles retries and failures

Built with:
- **Asynq Server** - Task processing server
- **Handler Registry** - Dynamic handler registration

### Redis

Redis serves as:
- Task queue storage
- Task state persistence
- Deduplication mechanism (unique constraints)

## Directory Structure

```
internal/
├── application/task/       # Application layer (CQRS pattern)
│   ├── service.go         # Service facade
│   ├── commands.go        # Command definitions
│   └── queries.go         # Query definitions
│
├── config/                 # Configuration management
│   └── config.go          # Config structs and loading
│
├── domain/task/           # Domain layer
│   ├── entity.go          # Task entity
│   └── repository.go      # Repository interface
│
├── infrastructure/        # Infrastructure layer
│   ├── queue/asynq/       # Asynq client/server wrappers
│   └── observability/     # Logging
│
├── interfaces/http/       # HTTP interface layer
│   ├── handler/           # HTTP handlers
│   ├── dto/               # Data transfer objects
│   ├── middleware/        # HTTP middleware
│   └── router.go          # Route definitions
│
└── worker/                # Worker implementation
    ├── base.go            # Base handler interface
    ├── registry.go        # Handler registry
    ├── middleware.go      # Worker middleware
    └── handlers/          # Task handlers
        └── demo/          # Demo handler implementation
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.25.1 |
| HTTP Framework | Gin |
| Task Queue | Asynq |
| Message Broker | Redis |
| Configuration | Viper |
| Logging | Zap |

## Data Flow

### Task Creation

```
1. Client sends POST /api/v1/tasks
2. API validates request
3. Task payload serialized to JSON
4. Asynq client enqueues task to Redis
5. API returns task ID
```

### Task Processing

```
1. Worker polls Redis queue
2. Worker fetches task
3. Handler deserializes payload
4. Handler processes task
5. On success: task marked complete
6. On failure: task retried or archived
```

## Queue Priorities

Tasks are processed based on queue priority weights:

| Queue | Priority Weight | Use Case |
|-------|-----------------|----------|
| critical | 10 | System-critical tasks |
| high | 5 | Time-sensitive tasks |
| default | 3 | Normal tasks |
| low | 1 | Background tasks |

Higher weight = more processing time allocation.

## Middleware

### API Middleware

- **Recovery** - Panic recovery
- **Logger** - Request logging
- **CORS** - Cross-origin requests
- **RequestID** - Request tracing

### Worker Middleware

- **Recovery** - Panic recovery
- **Logging** - Task execution logging

## Scalability

- **Horizontal scaling**: Run multiple API and Worker instances
- **Concurrency**: Configure worker concurrency per instance
- **Queue isolation**: Route tasks to specific queues
