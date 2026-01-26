package progress

import "time"

// Progress 表示任务执行进度
type Progress struct {
	TaskID      string            `json:"task_id"`
	Percentage  int32             `json:"percentage"`
	Stage       string            `json:"stage"`
	Message     string            `json:"message"`
	TimestampMs int64             `json:"timestamp_ms"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Event 表示进度事件（包含 Stream 元信息）
type Event struct {
	ID       string   `json:"id"`       // Redis Stream entry ID
	Progress Progress `json:"progress"` // 进度数据
}

// TaskCompleted 表示任务完成事件
type TaskCompleted struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"` // completed, failed, cancelled
	Message   string `json:"message,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// NewProgress 创建进度对象
func NewProgress(taskID string, percentage int32, stage, message string) *Progress {
	return &Progress{
		TaskID:      taskID,
		Percentage:  percentage,
		Stage:       stage,
		Message:     message,
		TimestampMs: time.Now().UnixMilli(),
	}
}

// StreamKey 生成 Redis Stream key
func StreamKey(taskID string) string {
	return "progress:" + taskID
}

// CompletionKey 生成任务完成通知的 key
func CompletionKey(taskID string) string {
	return "progress:done:" + taskID
}

// DefaultStreamOptions 默认 Stream 配置
type StreamOptions struct {
	MaxLen      int64         // Stream 最大长度
	TTL         time.Duration // Stream 过期时间
	ReadTimeout time.Duration // 读取超时
}

// DefaultOptions 返回默认配置
func DefaultOptions() StreamOptions {
	return StreamOptions{
		MaxLen:      1000,              // 保留最近 1000 条进度
		TTL:         1 * time.Hour,     // 1 小时后过期
		ReadTimeout: 30 * time.Second,  // 30 秒读取超时
	}
}
