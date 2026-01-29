# API Reference

Base URL: `http://localhost:8080`

## Tasks

### Create Task

Creates a new task and enqueues it for processing.

**Endpoint:** `POST /api/v1/tasks`

**Request Body:**

```json
{
  "type": "demo",
  "payload": {
    "message": "Hello",
    "count": 5
  },
  "queue": "default",
  "max_retries": 3,
  "timeout": "30s",
  "process_at": "2024-01-15T10:00:00Z",
  "unique": "1h",
  "metadata": {
    "user_id": "123"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| type | string | Yes | Task type (e.g., "demo") |
| payload | object | Yes | Task-specific payload |
| queue | string | No | Queue name (default: "default") |
| max_retries | int | No | Maximum retry attempts |
| timeout | string | No | Task timeout (e.g., "30s", "5m") |
| process_at | string | No | Scheduled execution time (RFC3339) |
| unique | string | No | Deduplication window (e.g., "1h") |
| metadata | object | No | Custom metadata key-value pairs |

**Response:** `201 Created`

```json
{
  "task_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "queue": "default",
  "status": "pending"
}
```

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 400 | INVALID_REQUEST | Invalid request body |
| 400 | INVALID_TASK_TYPE | Unknown task type |
| 400 | INVALID_PAYLOAD | Invalid payload format |
| 400 | INVALID_TIMEOUT | Invalid timeout format |
| 400 | INVALID_PROCESS_AT | Invalid process_at format |
| 400 | INVALID_UNIQUE | Invalid unique format |
| 500 | INTERNAL_ERROR | Server error |

---

### Get Task

Retrieves task information by ID.

**Endpoint:** `GET /api/v1/tasks/:id`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| queue | string | No | Queue name (default: "default") |

**Response:** `200 OK`

```json
{
  "id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "queue": "default",
  "type": "demo",
  "state": "active",
  "max_retry": 3,
  "retried": 0,
  "last_err": "",
  "next_process_at": "2024-01-15T10:00:00Z"
}
```

**Task States:**

| State | Description |
|-------|-------------|
| pending | Task is waiting to be processed |
| active | Task is currently being processed |
| scheduled | Task is scheduled for future execution |
| retry | Task is waiting for retry |
| archived | Task failed and was archived |
| completed | Task completed successfully |

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 404 | TASK_NOT_FOUND | Task not found |
| 500 | INTERNAL_ERROR | Server error |

---

### Cancel Task

Cancels a pending or scheduled task.

**Endpoint:** `POST /api/v1/tasks/:id/cancel`

**Response:** `200 OK`

```json
{
  "message": "task cancelled"
}
```

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 500 | CANCEL_FAILED | Failed to cancel task |

---

### Delete Task

Deletes a task from the queue.

**Endpoint:** `DELETE /api/v1/tasks/:id`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| queue | string | No | Queue name (default: "default") |

**Response:** `200 OK`

```json
{
  "message": "task deleted"
}
```

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 500 | DELETE_FAILED | Failed to delete task |

---

## Task Progress

### Get Latest Progress

Retrieves the latest progress for a task.

**Endpoint:** `GET /api/v1/tasks/:id/progress`

**Response:** `200 OK`

```json
{
  "progress": {
    "task_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
    "percentage": 50,
    "stage": "processing",
    "message": "Processing data...",
    "timestamp_ms": 1737884800000
  },
  "is_final": false,
  "stream_id": "1737884800000-0"
}
```

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 404 | PROGRESS_NOT_FOUND | No progress found for this task |
| 500 | PROGRESS_FETCH_ERROR | Server error |

---

### Stream Progress (SSE)

Subscribes to real-time progress updates via Server-Sent Events.

**Endpoint:** `GET /api/v1/tasks/:id/progress/stream`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| history | string | No | Set to "true" to include historical progress |
| start_id | string | No | Stream ID to start from ("0" for all history, "$" for new only) |

**Response:** `200 OK` (text/event-stream)

```
event: progress
data: {"task_id":"xxx","percentage":30,"stage":"processing","message":"Processing...","timestamp_ms":1737884800000}

event: progress
data: {"task_id":"xxx","percentage":100,"stage":"completed","message":"Done","timestamp_ms":1737884810000}

event: done
data: {"task_id":"xxx","status":"completed"}
```

**Event Types:**

| Event | Description |
|-------|-------------|
| progress | Progress update |
| history | Historical progress (when history=true) |
| done | Task completed/failed/cancelled |
| error | Error occurred |

**Example (curl):**

```bash
curl -N "http://localhost:8080/api/v1/tasks/xxx/progress/stream"
```

**Example (JavaScript):**

```javascript
const es = new EventSource(`/api/v1/tasks/${taskId}/progress/stream`);
es.addEventListener('progress', (e) => {
    const data = JSON.parse(e.data);
    console.log(`[${data.percentage}%] ${data.message}`);
});
es.addEventListener('done', () => es.close());
```

---

### Stream Multiple Progress (SSE)

Subscribes to progress updates for multiple tasks simultaneously.

**Endpoint:** `GET /api/v1/progress/stream`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| task_ids | string | Yes | Comma-separated task IDs (max 10) |

**Response:** `200 OK` (text/event-stream)

```
event: progress
data: {"task_id":"id1","progress":{"percentage":30,...}}

event: progress
data: {"task_id":"id2","progress":{"percentage":50,...}}
```

---

### Get Progress History

Retrieves historical progress entries for a task.

**Endpoint:** `GET /api/v1/tasks/:id/progress/history`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| start_id | string | No | Stream ID to start from (default: "-") |

**Response:** `200 OK`

```json
{
  "task_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "count": 5,
  "history": [
    {
      "stream_id": "1737884800000-0",
      "progress": {"percentage": 10, "stage": "init", "message": "Starting..."},
      "is_final": false
    },
    {
      "stream_id": "1737884810000-0",
      "progress": {"percentage": 100, "stage": "completed", "message": "Done"},
      "is_final": true,
      "status": "completed"
    }
  ]
}
```

---

### Get Progress Info

Retrieves metadata about a task's progress stream.

**Endpoint:** `GET /api/v1/tasks/:id/progress/info`

**Response:** `200 OK`

```json
{
  "task_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
  "has_progress": true,
  "length": 5,
  "first_entry": "1737884800000-0",
  "last_entry": "1737884810000-0"
}
```

---

## Queues

### Get Queue Stats

Retrieves statistics for all queues or a specific queue.

**Endpoint:** `GET /api/v1/queues/stats`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| queue | string | No | Specific queue name (returns all if omitted) |

**Response:** `200 OK`

```json
[
  {
    "queue": "default",
    "pending": 10,
    "active": 2,
    "scheduled": 5,
    "retry": 1,
    "archived": 0,
    "completed": 100
  },
  {
    "queue": "critical",
    "pending": 0,
    "active": 1,
    "scheduled": 0,
    "retry": 0,
    "archived": 0,
    "completed": 50
  }
]
```

**Error Responses:**

| Code | Error Code | Description |
|------|------------|-------------|
| 500 | STATS_FAILED | Failed to retrieve stats |

---

## Health Checks

### Health

General health check endpoint.

**Endpoint:** `GET /health`

**Response:** `200 OK`

```json
{
  "status": "healthy",
  "timestamp": "2026-01-29T12:00:00Z",
  "services": {
    "redis": "healthy"
  }
}
```

---

### Ready

Readiness check (verifies Redis connection).

**Endpoint:** `GET /ready`

**Response:** `200 OK`

```json
{
  "status": "ready"
}
```

**Error Response:** `503 Service Unavailable`

```json
{
  "status": "not ready",
  "reason": "redis unavailable"
}
```

---

### Live

Liveness check endpoint.

**Endpoint:** `GET /live`

**Response:** `200 OK`

```json
{
  "status": "alive"
}
```

---

## Error Response Format

All error responses follow this format:

```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| error | string | Human-readable error message |
| code | string | Machine-readable error code |
| details | object | Additional error details (optional) |
