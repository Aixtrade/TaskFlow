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
  "status": "ok"
}
```

---

### Ready

Readiness check (verifies Redis connection).

**Endpoint:** `GET /ready`

**Response:** `200 OK`

```json
{
  "status": "ok"
}
```

**Error Response:** `503 Service Unavailable`

```json
{
  "status": "error",
  "message": "redis connection failed"
}
```

---

### Live

Liveness check endpoint.

**Endpoint:** `GET /live`

**Response:** `200 OK`

```json
{
  "status": "ok"
}
```

---

## Metrics

### Prometheus Metrics

**Endpoint:** `GET /metrics`

Returns Prometheus-formatted metrics including:
- HTTP request counts and latencies
- Task processing counts and durations
- Queue depths
- Go runtime metrics

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
