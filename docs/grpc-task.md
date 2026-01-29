# GRPC 任务类型使用指南（Python 示例）

本文说明如何在 TaskFlow 中使用 `grpc_task` 任务类型，并用 Python 实现一个符合规范的 gRPC 服务端。

## 概览

`grpc_task` 由 `internal/worker/handlers/grpc_task/handler.go` 处理，核心流程：

1. 解析 `GRPCTaskPayload`
2. 校验服务名是否在配置中存在
3. 通过 `ClientManager` 获取 gRPC 客户端
4. 调用 `TaskExecutorService.ExecuteTask`（流式）
5. 处理 `Progress`/`TaskResult`/`ErrorDetail`

## 任务类型与 Payload

- 任务类型：`grpc_task`
- Payload 结构：`pkg/payload/grpc_task.go` 中的 `GRPCTaskPayload`

```json
{
  "service": "llm",
  "method": "chat",
  "data": {
    "prompt": "hello"
  },
  "options": {
    "timeout_ms": 600000,
    "enable_progress": true,
    "progress_interval_ms": 1000
  }
}
```

字段说明：

- `service`：服务名（必填），必须在 `grpc_services.services` 配置中存在
- `method`：任务方法名，会映射到 gRPC 请求里的 `task_type`
- `data`：业务数据，映射到 `ExecuteTaskRequest.payload`（`google.protobuf.Struct`）
- `options`：执行选项，覆盖默认超时与进度设置

## 配置 gRPC 服务

在配置文件中启用并注册服务：

```yaml
grpc_services:
  enabled: true
  services:
    llm:
      address: "llm-service:50051"
      timeout: 600s
      health_check_interval: 30s
      max_retries: 3
      retry_delay: 1s
  defaults:
    timeout: 300s
    health_check_interval: 30s
    max_retries: 3
    retry_delay: 1s
```

服务名 `llm` 需要与 Payload 的 `service` 字段一致。

## gRPC 接口规范

协议文件：`api/proto/grpc_task/v1/task.proto`

服务定义：

```proto
service TaskExecutorService {
  rpc ExecuteTask(ExecuteTaskRequest) returns (stream ExecuteTaskResponse);
  rpc CancelTask(CancelTaskRequest) returns (CancelTaskResponse);
  rpc HealthCheck(HealthCheckRequest) returns (HealthCheckResponse);
}
```

关键消息：

- `ExecuteTaskRequest`
  - `task_id`：TaskFlow 任务 ID
  - `task_type`：来自 payload 的 `method`
  - `payload`：业务数据（Struct）
  - `metadata`：包含 `service`、`queue`、`retry_count`、`max_retry`
  - `options`：超时与进度设置

- `ExecuteTaskResponse`
  - `progress`：进度信息（可多次发送）
  - `result`：最终结果（建议只发送一次）
  - `error`：错误详情（发送后将视为失败）

- `ErrorDetail.retryable`
  - `false`：TaskFlow 将停止重试（等价于 `asynq.SkipRetry`）
  - `true`：TaskFlow 会按重试策略重试

## Python 服务端实现步骤

以下以 Python 版本实现 `TaskExecutorService` 为例。

### 1. 生成 Python proto

示例目录已迁移到 `examples/python_service`。

```bash
python3 -m venv .venv
source .venv/bin/activate
python3 -m pip install -r examples/python_service/requirements.txt
bash examples/python_service/generate_proto.sh
```

### 2. 实现服务端

参考 `examples/python_service/server.py`，核心结构：

```python
import grpc
from concurrent import futures
from google.protobuf import struct_pb2
import grpc_task.v1.task_pb2 as pb
import grpc_task.v1.task_pb2_grpc as pb_grpc

class TaskExecutorServicer(pb_grpc.TaskExecutorServiceServicer):
    async def ExecuteTask(self, request, context):
        task_id = request.task_id
        method = request.task_type

        # 发送进度
        yield pb.ExecuteTaskResponse(
            progress=pb.Progress(
                task_id=task_id,
                percentage=10,
                stage="init",
                message="starting",
                timestamp_ms=int(time.time() * 1000),
            )
        )

        # 业务处理完成后，发送结果
        result_data = struct_pb2.Struct()
        result_data.update({"answer": "ok"})
        yield pb.ExecuteTaskResponse(
            result=pb.TaskResult(
                task_id=task_id,
                status=pb.TASK_STATUS_COMPLETED,
                data=result_data,
                duration_ms=1200,
            )
        )

    async def CancelTask(self, request, context):
        return pb.CancelTaskResponse(success=True, message="cancel requested")

    async def HealthCheck(self, request, context):
        return pb.HealthCheckResponse(
            status=pb.HEALTH_STATUS_HEALTHY,
            message="ok",
            details={"handlers": "demo"},
        )

async def serve(port=50051):
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    pb_grpc.add_TaskExecutorServiceServicer_to_server(TaskExecutorServicer(), server)
    server.add_insecure_port(f"[::]:{port}")
    await server.start()
    await server.wait_for_termination()
```

实现要点：

- `ExecuteTask` 必须是流式响应，允许发送多个 `progress`，最终发送 `result` 或 `error`
- `CancelTask` 用于任务取消（TaskFlow 的 `CancelTask` API 会触发调用）
- `HealthCheck` 用于服务健康状态监测

### 3. 启动服务并配置 TaskFlow

确保服务启动并可访问后，在配置中注册服务地址，并将任务 `payload.service` 指向该名称。

## 任务创建示例

通过 API 创建 `grpc_task`：

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "grpc_task",
    "payload": {
      "service": "llm",
      "method": "chat",
      "data": {
        "prompt": "hello"
      },
      "options": {
        "timeout_ms": 600000,
        "enable_progress": true,
        "progress_interval_ms": 1000
      }
    },
    "queue": "default",
    "max_retries": 3
  }'
```

## 实时进度订阅

TaskFlow 支持通过 SSE (Server-Sent Events) 实时订阅任务进度。进度数据存储在 Redis Streams 中，支持历史回放。

### 快速示例

```bash
# SSE 订阅进度
curl -N http://localhost:8080/api/v1/tasks/{task_id}/progress/stream

# 获取最新进度
curl http://localhost:8080/api/v1/tasks/{task_id}/progress
```

**详细 API 文档请参阅 [api.md](api.md#task-progress)**

### 客户端示例

**JavaScript:**

```javascript
const es = new EventSource(`/api/v1/tasks/${taskId}/progress/stream`);
es.addEventListener('progress', (e) => {
    const data = JSON.parse(e.data);
    console.log(`[${data.percentage}%] ${data.message}`);
});
es.addEventListener('done', () => es.close());
```

**Python:**

```python
import sseclient, requests, json

url = f"http://localhost:8080/api/v1/tasks/{task_id}/progress/stream"
for event in sseclient.SSEClient(requests.get(url, stream=True)).events():
    if event.event == 'progress':
        print(json.loads(event.data))
    elif event.event == 'done':
        break
```

## 运行时行为与错误处理

- `service` 不存在：任务直接 `SkipRetry`
- gRPC 服务不健康：返回错误触发重试
- `ErrorDetail.retryable=false`：任务不再重试
- `TaskResult.status=FAILED/CANCELLED`：TaskFlow 视为失败

## 关联文件

- `internal/worker/handlers/grpc_task/handler.go` - gRPC 任务处理器
- `pkg/payload/grpc_task.go` - Payload 结构定义
- `api/proto/grpc_task/v1/task.proto` - gRPC 协议定义
- `examples/python_service/server.py` - Python 服务端示例
- `pkg/progress/` - 进度发布/订阅模块
  - `types.go` - 进度数据结构
  - `publisher.go` - 发布到 Redis Stream
  - `subscriber.go` - 从 Redis Stream 订阅
- `internal/interfaces/http/handler/progress_handler.go` - SSE 端点处理器
