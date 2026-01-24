# TaskFlow

[English](README.md)

基于 Go 构建的分布式任务队列系统，使用 Redis 和 Asynq 实现可靠的后台任务处理。

## 特性

- **RESTful API** - 用于任务管理的 HTTP API（创建、查询、取消、删除）
- **分布式 Worker** - 可扩展的 Worker 池，支持可配置的并发数
- **优先级队列** - 支持 critical、high、default、low 优先级队列
- **定时任务** - 支持任务定时执行
- **任务去重** - 唯一性约束防止重复处理
- **重试机制** - 自动重试，支持配置最大重试次数
- **可观测性** - 内置 Prometheus 指标和结构化日志
- **健康检查** - 健康、就绪、存活检查端点

## 快速开始

### 环境要求

- Go 1.21+
- Redis 6.0+
- Docker（可选）

### 安装

```bash
# 克隆仓库
git clone https://github.com/Aixtrade/TaskFlow.git
cd TaskFlow

# 安装依赖
make deps

# 构建二进制文件
make build
```

### 使用 Docker 运行

```bash
# 启动 Redis
make redis-up

# 或使用 Docker Compose 启动完整环境
make docker-up
```

### 本地运行

1. 启动 Redis：
```bash
make redis-up
```

2. 启动 API 服务：
```bash
make run-api
```

3. 启动 Worker 服务（在另一个终端）：
```bash
make run-server
```

### 创建第一个任务

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

## 项目结构

```
TaskFlow/
├── cmd/
│   ├── api/           # API 服务入口
│   └── server/        # Worker 服务入口
├── configs/           # 配置文件
├── deployments/       # Docker 和部署文件
├── docs/              # 文档
├── internal/
│   ├── application/   # 应用服务 (CQRS)
│   ├── config/        # 配置加载
│   ├── domain/        # 领域实体和接口
│   ├── infrastructure/# 外部依赖（Redis、Asynq）
│   ├── interfaces/    # HTTP 处理器和 DTO
│   └── worker/        # 任务处理器和注册表
└── pkg/
    ├── errors/        # 通用错误类型
    ├── payload/       # 任务 Payload 定义
    └── tasktype/      # 任务类型常量
```

## 配置

TaskFlow 使用 YAML 配置文件，支持环境变量覆盖：

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

redis:
  addr: localhost:6379
  password: ""
  db: 0

queues:
  critical: 10
  high: 5
  default: 3
  low: 1
```

环境变量使用 `TASKFLOW_` 前缀：
- `TASKFLOW_REDIS_ADDR`
- `TASKFLOW_SERVER_HTTP_PORT`
- 等等

## 文档

- [系统架构](docs/architecture.md) - 系统设计和组件说明
- [API 文档](docs/api.md) - REST API 接口文档
- [创建任务](docs/creating-tasks.md) - 自定义任务开发指南

## 开发

```bash
# 运行测试
make test

# 运行测试并生成覆盖率报告
make test-coverage

# 运行代码检查
make lint

# 清理构建产物
make clean
```

## 监控

- **指标**: 访问 `/metrics`（Prometheus 格式）
- **健康检查**: `GET /health`
- **就绪检查**: `GET /ready`
- **存活检查**: `GET /live`
- **Asynqmon 控制台**: 运行 `make asynqmon` 启动 Web 控制台

## 许可证

MIT License
