# Creating Tasks

This guide explains how to create custom task types in TaskFlow.

## Overview

Creating a new task type involves four steps:

1. Define the task type constant
2. Define the payload struct
3. Implement the handler
4. Register the handler

## Step 1: Define Task Type

Add a new task type constant in `pkg/tasktype/types.go`:

```go
package tasktype

type Type string

const (
    Demo  Type = "demo"
    Email Type = "email"  // New task type
)

func (t Type) String() string {
    return string(t)
}

func (t Type) IsValid() bool {
    switch t {
    case Demo, Email:
        return true
    }
    return false
}

var AllTypes = []Type{
    Demo,
    Email,
}
```

## Step 2: Define Payload

Create a new file in `pkg/payload/` for your task payload:

```go
// pkg/payload/email.go
package payload

type EmailPayload struct {
    To      string   `json:"to"`
    Subject string   `json:"subject"`
    Body    string   `json:"body"`
    CC      []string `json:"cc,omitempty"`
}
```

**Payload Guidelines:**
- Use JSON tags for serialization
- Include validation tags if needed
- Keep payloads serializable (no functions, channels, etc.)
- Use `omitempty` for optional fields

## Step 3: Implement Handler

Create a new handler in `internal/worker/handlers/`:

```go
// internal/worker/handlers/email/handler.go
package email

import (
    "context"
    "fmt"

    "github.com/hibiken/asynq"
    "go.uber.org/zap"

    "github.com/Aixtrade/TaskFlow/internal/worker"
    "github.com/Aixtrade/TaskFlow/pkg/payload"
    "github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Handler struct {
    *worker.BaseHandler
    // Add dependencies here (e.g., email client)
}

func NewHandler(logger *zap.Logger) *Handler {
    return &Handler{
        BaseHandler: worker.NewBaseHandler(logger),
    }
}

// Type returns the task type this handler processes
func (h *Handler) Type() string {
    return tasktype.Email.String()
}

// ProcessTask handles the email task
func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    taskID := worker.GetTaskID(ctx)
    h.LogTaskStart(h.Type(), taskID)

    // Unmarshal the payload
    p, err := worker.UnmarshalPayload[payload.EmailPayload](task)
    if err != nil {
        h.LogTaskError(h.Type(), taskID, err)
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }

    // Process the task
    h.Logger().Info("sending email",
        zap.String("to", p.To),
        zap.String("subject", p.Subject),
    )

    // Check for cancellation
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
    }

    // TODO: Implement email sending logic here
    // err = h.emailClient.Send(p.To, p.Subject, p.Body)

    h.LogTaskComplete(h.Type(), taskID)
    return nil
}
```

## Step 4: Register Handler

Register the handler in `cmd/server/main.go`:

```go
package main

import (
    // ... other imports
    "github.com/Aixtrade/TaskFlow/internal/worker/handlers/demo"
    "github.com/Aixtrade/TaskFlow/internal/worker/handlers/email"  // Import new handler
)

func main() {
    // ... initialization code

    registry := worker.NewRegistry(logger)

    // Register handlers
    registry.Register(demo.NewHandler(logger))
    registry.Register(email.NewHandler(logger))  // Register new handler

    // ... rest of the code
}
```

## Complete Example: Image Processing Task

Here's a complete example of creating an image processing task:

### 1. Task Type (`pkg/tasktype/types.go`)

```go
const (
    Demo         Type = "demo"
    ImageProcess Type = "image:process"
)
```

### 2. Payload (`pkg/payload/image.go`)

```go
package payload

type ImageProcessPayload struct {
    ImageURL   string `json:"image_url"`
    OutputPath string `json:"output_path"`
    Width      int    `json:"width,omitempty"`
    Height     int    `json:"height,omitempty"`
    Quality    int    `json:"quality,omitempty"`
}

type ImageProcessResult struct {
    TaskID     string `json:"task_id"`
    OutputURL  string `json:"output_url"`
    FileSize   int64  `json:"file_size"`
}
```

### 3. Handler (`internal/worker/handlers/image/handler.go`)

```go
package image

import (
    "context"
    "fmt"

    "github.com/hibiken/asynq"
    "go.uber.org/zap"

    "github.com/Aixtrade/TaskFlow/internal/worker"
    "github.com/Aixtrade/TaskFlow/pkg/payload"
    "github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Handler struct {
    *worker.BaseHandler
}

func NewHandler(logger *zap.Logger) *Handler {
    return &Handler{
        BaseHandler: worker.NewBaseHandler(logger),
    }
}

func (h *Handler) Type() string {
    return tasktype.ImageProcess.String()
}

func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    taskID := worker.GetTaskID(ctx)
    h.LogTaskStart(h.Type(), taskID)

    p, err := worker.UnmarshalPayload[payload.ImageProcessPayload](task)
    if err != nil {
        h.LogTaskError(h.Type(), taskID, err)
        return fmt.Errorf("failed to unmarshal payload: %w", err)
    }

    h.Logger().Info("processing image",
        zap.String("url", p.ImageURL),
        zap.Int("width", p.Width),
        zap.Int("height", p.Height),
    )

    // Check context for cancellation
    if err := ctx.Err(); err != nil {
        return err
    }

    // TODO: Implement image processing logic
    // 1. Download image from p.ImageURL
    // 2. Resize to p.Width x p.Height
    // 3. Save to p.OutputPath with p.Quality

    h.LogTaskComplete(h.Type(), taskID)
    return nil
}
```

### 4. Register Handler

```go
registry.Register(image.NewHandler(logger))
```

### 5. Create Task via API

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "image:process",
    "payload": {
      "image_url": "https://example.com/image.jpg",
      "output_path": "/processed/image.jpg",
      "width": 800,
      "height": 600,
      "quality": 85
    },
    "queue": "default",
    "timeout": "5m",
    "max_retries": 3
  }'
```

## Helper Functions

The `worker` package provides useful helper functions:

```go
// Get task metadata from context
taskID := worker.GetTaskID(ctx)
retryCount := worker.GetRetryCount(ctx)
maxRetry := worker.GetMaxRetry(ctx)
queueName := worker.GetQueueName(ctx)

// Unmarshal payload with type safety
payload, err := worker.UnmarshalPayload[YourPayloadType](task)
```

## Error Handling

Return errors to trigger retries:

```go
func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    // Permanent failure - don't retry
    if err := validate(payload); err != nil {
        return fmt.Errorf("validation failed: %w", asynq.SkipRetry)
    }

    // Temporary failure - will retry
    if err := process(payload); err != nil {
        return fmt.Errorf("processing failed: %w", err)
    }

    return nil
}
```

## Best Practices

1. **Idempotency**: Design handlers to be idempotent (safe to run multiple times)
2. **Timeout Handling**: Respect context cancellation for long-running tasks
3. **Logging**: Use structured logging with task ID for traceability
4. **Error Messages**: Return descriptive errors for debugging
5. **Payload Validation**: Validate payload early in the handler
6. **Resource Cleanup**: Clean up resources on cancellation
