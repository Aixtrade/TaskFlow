# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TaskFlow is a distributed task queue system built with Go, using Asynq (Redis-based) as the message broker. It provides a RESTful API for task management and a worker server for task processing.

## Commands

```bash
# Build
make build              # Build both api and server binaries
make build-api          # Build API server only
make build-server       # Build worker server only

# Development
make run-api            # Run API server (uses config.dev.yaml)
make run-server         # Run worker server (uses config.dev.yaml)
make redis-up           # Start Redis container
make asynqmon           # Start Asynqmon UI at http://localhost:8081

# Testing
make test               # Run all tests with race detection and coverage
make test-coverage      # Generate HTML coverage report

# Code Quality
make lint               # Run golangci-lint

# Docker
make docker-up          # Start full stack (Redis, API, Worker)
make docker-down        # Stop Docker Compose stack
```

## Architecture

### Two Main Services
- **API Server** (`cmd/api/main.go`): RESTful API on port 8080 for task CRUD operations
- **Worker Server** (`cmd/server/main.go`): Background processor that executes queued tasks

### Layered Structure
```
internal/
├── application/task/    # CQRS: commands (Create, Cancel, Delete) and queries
├── domain/task/         # Task entity, repository interface
├── infrastructure/      # Asynq client/server wrappers, logging
├── interfaces/http/     # Gin router, handlers, DTOs, middleware
└── worker/              # Handler registry, base handler, middleware

pkg/
├── tasktype/            # Task type constants (Demo, Email, etc.)
├── payload/             # Task payload structs
└── errors/              # Shared error types
```

### Handler Registry Pattern
Workers use a registry to dynamically register task handlers:
```go
registry := worker.NewRegistry(logger)
registry.Register(demo.NewHandler(logger))
registry.SetupServer(server)
```

## Creating a New Task Type

1. **Define type** in `pkg/tasktype/types.go`:
   ```go
   const Email Type = "email"
   ```
   Update `IsValid()` and `AllTypes` accordingly.

2. **Define payload** in `pkg/payload/email.go`:
   ```go
   type EmailPayload struct {
       To      string `json:"to"`
       Subject string `json:"subject"`
   }
   ```

3. **Implement handler** in `internal/worker/handlers/email/handler.go`:
   ```go
   func (h *Handler) Type() string { return tasktype.Email.String() }
   func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
       payload, err := worker.UnmarshalPayload[payload.EmailPayload](task)
       // process...
   }
   ```

4. **Register** in `cmd/server/main.go`:
   ```go
   registry.Register(email.NewHandler(logger))
   ```

## Configuration

YAML config in `configs/`. Environment variables override with `TASKFLOW_` prefix:
- `TASKFLOW_SERVER_HTTP_PORT=8081`
- `TASKFLOW_REDIS_ADDR=redis:6379`
- `TASKFLOW_SERVER_WORKER_CONCURRENCY=20`

## Key Patterns

- **CQRS**: Application layer separates commands from queries
- **Error handling**: Use `asynq.SkipRetry` for permanent errors, return error for retry
- **Context utilities**: `worker.GetTaskID(ctx)`, `worker.GetRetryCount(ctx)`
- **Queue priorities**: critical(10), high(5), default(3), low(1) - higher weight = more processing time

## API Endpoints

- `POST /api/v1/tasks` - Create task
- `GET /api/v1/tasks/:id` - Get task info
- `POST /api/v1/tasks/:id/cancel` - Cancel task
- `DELETE /api/v1/tasks/:id` - Delete task
- `GET /api/v1/queues/stats` - Queue statistics
- `GET /health`, `/ready`, `/live` - Health checks
