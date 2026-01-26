package payload

// GRPCTaskPayload 定义 gRPC 流式任务的输入结构
// 可用于调用任何语言实现的 gRPC 服务（Python、Java、Node.js、Rust 等）
type GRPCTaskPayload struct {
	// Service 目标服务名称（必填），如 "llm", "trading", "data"
	Service string `json:"service"`

	// Method 调用的方法名（可选），如 "chat", "backtest"
	Method string `json:"method,omitempty"`

	// Data 业务数据
	Data map[string]interface{} `json:"data"`

	// Options 任务执行选项（可选）
	Options *GRPCTaskOptions `json:"options,omitempty"`
}

// GRPCTaskOptions 任务执行选项，用于覆盖默认配置
type GRPCTaskOptions struct {
	// TimeoutMs 超时时间（毫秒），覆盖服务默认超时
	TimeoutMs *int `json:"timeout_ms,omitempty"`

	// EnableProgress 是否启用进度报告
	EnableProgress *bool `json:"enable_progress,omitempty"`

	// ProgressIntervalMs 进度报告间隔（毫秒）
	ProgressIntervalMs *int `json:"progress_interval_ms,omitempty"`
}

// GRPCTaskResult 定义 gRPC 流式任务的输出结构
type GRPCTaskResult struct {
	// TaskID 任务ID
	TaskID string `json:"task_id"`

	// Service 处理该任务的服务名
	Service string `json:"service"`

	// Method 调用的方法名
	Method string `json:"method,omitempty"`

	// Status 任务状态: completed, failed, cancelled
	Status string `json:"status"`

	// Data 返回数据
	Data map[string]interface{} `json:"data,omitempty"`

	// DurationMs 执行耗时（毫秒）
	DurationMs int64 `json:"duration_ms"`

	// Error 错误信息（如果失败）
	Error *GRPCTaskError `json:"error,omitempty"`
}

// GRPCTaskError 任务错误信息
type GRPCTaskError struct {
	// Code 错误码
	Code string `json:"code"`

	// Message 错误消息
	Message string `json:"message"`

	// Retryable 是否可重试
	Retryable bool `json:"retryable"`
}

// Validate 验证 payload 是否有效
func (p *GRPCTaskPayload) Validate() error {
	if p.Service == "" {
		return &ValidationError{Field: "service", Message: "service is required"}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}
