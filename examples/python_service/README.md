# Python gRPC Task Executor Service

符合 TaskFlow `grpc_task` 规范的 Python gRPC 服务示例。

## 快速开始

### 使用 Docker Compose（推荐）

```bash
# 进入目录
cd examples/python_service

# 构建并启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

服务将在 `localhost:50051` 启动。

**注意**: Docker Compose 使用项目根目录作为 build context，以便访问 `api/proto` 文件。

### 本地开发

1. **安装依赖**

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

2. **生成 Proto 文件**

```bash
bash generate_proto.sh
```

3. **启动服务**

```bash
python server.py --port 50051
```

## 配置 TaskFlow

在 TaskFlow 配置文件中注册此服务：

```yaml
grpc_services:
  enabled: true
  services:
    python:  # 服务名称
      address: "localhost:50051"  # Docker 中使用: python-grpc-service:50051
      timeout: 600s
      pool_size: 10
      health_check_interval: 30s
```

## 示例任务

### Demo 任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "grpc_task",
    "payload": {
      "service": "python",
      "method": "demo",
      "data": {
        "message": "Hello TaskFlow",
        "count": 5
      }
    },
    "queue": "default"
  }'
```

### Chat 任务（模拟 LLM）

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "grpc_task",
    "payload": {
      "service": "python",
      "method": "chat",
      "data": {
        "prompt": "你好，请介绍一下自己",
        "max_tokens": 100
      },
      "options": {
        "timeout_ms": 600000,
        "enable_progress": true
      }
    },
    "queue": "default"
  }'
```

### Backtest 任务（模拟回测）

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "type": "grpc_task",
    "payload": {
      "service": "python",
      "method": "backtest",
      "data": {
        "strategy_id": "momentum_v1",
        "start_date": "2024-01-01",
        "end_date": "2024-12-31"
      }
    },
    "queue": "default"
  }'
```

## 添加自定义 Handler

在 `server.py` 中注册新的 handler：

```python
async def my_custom_handler(
    request: Any,
    context: grpc.aio.ServicerContext,
    task_state: dict
) -> AsyncIterator[Any]:
    task_id = request.task_id
    payload = payload_to_dict(request.payload)

    # 你的业务逻辑
    for i in range(10):
        if task_state.get("cancelled"):
            return

        await asyncio.sleep(0.5)

        yield create_progress(
            task_id=task_id,
            percentage=int((i + 1) / 10 * 100),
            stage="processing",
            message=f"Step {i + 1}"
        )

# 在 serve() 函数中注册
servicer.register_handler("my_method", my_custom_handler)
```

## 健康检查

```bash
# 使用 grpcurl
grpcurl -plaintext localhost:50051 grpc_task.v1.TaskExecutorService/HealthCheck
```

## 架构说明

- **动态 Handler 注册**: 通过 `register_handler()` 注册任务处理器
- **流式响应**: 支持发送多个进度更新
- **任务取消**: 支持优雅取消正在运行的任务
- **错误处理**: 区分可重试和不可重试错误
- **优雅关闭**: 处理 SIGINT/SIGTERM 信号

## 目录结构

```
examples/python_service/
├── server.py              # 主服务实现
├── requirements.txt       # Python 依赖
├── Dockerfile            # Docker 镜像定义
├── docker-compose.yml    # Docker Compose 配置
├── generate_proto.sh     # Proto 生成脚本
├── .dockerignore         # Docker 忽略文件
└── README.md             # 本文档
```

## 参考文档

- [GRPC 任务类型使用指南](../../docs/grpc-task.md)
- [Proto 定义](../../api/proto/grpc_task/v1/task.proto)
