# AGENTS.md

本文件为在本仓库中工作的智能体提供执行与编码规范。

## 基本信息

- 项目：TaskFlow（Go 1.25.1）
- 结构：分层 + CQRS（application / domain / infrastructure / interfaces / worker）
- 关键入口：`cmd/api/main.go`、`cmd/server/main.go`
- 配置：`configs/*.yaml`，支持 `TASKFLOW_` 前缀环境变量覆盖

## 构建 / 运行 / 测试 / 质量

### 依赖与构建

- 下载依赖：`make deps`
- 整理依赖：`make tidy`
- 构建全部：`make build`
- 构建 API：`make build-api`
- 构建 Worker：`make build-server`

### 本地运行

- 运行 API：`make run-api`（使用 `configs/config.dev.yaml`）
- 运行 Worker：`make run-server`（使用 `configs/config.dev.yaml`）

### 测试

- 全量测试（含 race + coverage）：`make test`
- 生成覆盖率报告：`make test-coverage`

### 单测（单个包 / 单个用例）

- 单个包：`go test ./internal/application/task`
- 单个用例：`go test ./internal/application/task -run ^TestName$`
- 全仓筛选单测：`go test ./... -run ^TestName$`
- 若需禁用缓存：`go test ./internal/application/task -run ^TestName$ -count=1`

### 代码质量

- Lint：`make lint`（golangci-lint）

### Docker / Redis

- 启动 Redis：`make redis-up`
- 关闭 Redis：`make redis-down`
- Docker 全栈：`make docker-up` / `make docker-down`
- Asynqmon：`make asynqmon`（UI 端口 8081）

### Proto 相关

- 生成 Protobuf：`make proto-gen`
- 清理生成文件：`make proto-clean`

## 代码风格与约定

### 格式化与工具

- 必须通过 `gofmt`（不要手工调整格式）
- 建议使用 `goimports` 自动整理 import（保持分组与排序）
- 仅在文件已有非 ASCII 时引入非 ASCII；否则保持 ASCII

### Import 分组

- 标准库、第三方、项目内包分为 3 组
- 组内按字母序排列
- 项目内包以 `github.com/Aixtrade/TaskFlow/...` 路径引用

### 命名规范

- 包名：小写单词，避免下划线
- 类型/结构体：`UpperCamelCase`
- 函数/方法：`UpperCamelCase`（导出）或 `lowerCamelCase`（非导出）
- 变量：短小且语义明确；`ctx`、`cfg`、`err` 按 Go 习惯使用
- 错误变量：`ErrXxx` 作为哨兵错误（`errors.New`）

### 日志与可观测性

- 使用 `zap.Logger`（见 `internal/infrastructure/observability/logging`）
- 结构化字段：`zap.String` / `zap.Int` 等
- 关键流程打印 Info，异常打印 Error/Fatal

### 错误处理

- 尽量返回原始错误或使用 `fmt.Errorf("...: %w", err)` 包装
- 通过 `errors.Is` / `errors.As` 判断错误类型
- 领域错误在 `internal/domain/task/errors.go` 中集中管理
- 通用错误与包装类型在 `pkg/errors` 中管理
- Worker 任务中：可用 `asynq.SkipRetry` 表示不可重试错误

### DTO / JSON

- DTO 放在 `internal/interfaces/http/dto`
- JSON 字段使用 snake_case（见现有 struct tag）
- 输出时间格式通常为 RFC3339（示例见 `TaskInfo.NextProcessAt`）

### CQRS 层次

- Command / Query 定义在 `internal/application/task`
- Handler 仅负责解析请求、验证参数、调用 Service
- Service 处理业务逻辑与依赖调用
- Domain 实体与验证逻辑放在 `internal/domain`

### Worker 任务

- Handler 放在 `internal/worker/handlers/<task>`
- `Type()` 返回 `tasktype.Xxx.String()`
- 使用 `worker.UnmarshalPayload[T]` 解析 payload
- 注册在 `cmd/server/main.go`

### 配置与环境

- 配置入口：`internal/config`（Viper）
- 默认使用 `configs/config.dev.yaml` 进行本地开发
- 环境变量以 `TASKFLOW_` 前缀覆盖

## 目录约定

- `cmd/`：应用入口
- `internal/application/`：应用服务、命令与查询
- `internal/domain/`：领域模型与接口
- `internal/infrastructure/`：外部依赖实现（Redis/Asynq/日志/监控）
- `internal/interfaces/`：HTTP 接口、路由、DTO、中间件
- `internal/worker/`：任务处理器、注册表、中间件
- `pkg/`：可复用的公共包

## 变更建议

- 新增任务类型请同步修改：
  - `pkg/tasktype/types.go`
  - `pkg/payload/<task>.go`
  - `internal/worker/handlers/<task>/handler.go`
  - `cmd/server/main.go`（注册 handler）

## Cursor / Copilot 规则

- 未发现 `.cursor/rules/`、`.cursorrules` 或 `.github/copilot-instructions.md`
