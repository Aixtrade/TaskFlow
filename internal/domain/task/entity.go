package task

import (
	"encoding/json"
	"time"

	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusScheduled Status = "scheduled"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
	StatusRetrying  Status = "retrying"
)

type Task struct {
	ID          string         `json:"id"`
	Type        tasktype.Type  `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Status      Status         `json:"status"`
	Queue       string         `json:"queue"`
	Priority    int            `json:"priority"`
	MaxRetries  int            `json:"max_retries"`
	Retried     int            `json:"retried"`
	Timeout     time.Duration  `json:"timeout"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	ScheduledAt time.Time      `json:"scheduled_at,omitempty"`
	StartedAt   time.Time      `json:"started_at,omitempty"`
	CompletedAt time.Time      `json:"completed_at,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func NewTask(taskType tasktype.Type, payload any) (*Task, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &Task{
		Type:       taskType,
		Payload:    payloadBytes,
		Status:     StatusPending,
		Queue:      taskType.Queue(),
		MaxRetries: 3,
		Timeout:    30 * time.Minute,
		CreatedAt:  time.Now(),
		Metadata:   make(map[string]string),
	}, nil
}

func (t *Task) SetScheduledAt(at time.Time) {
	t.ScheduledAt = at
	t.Status = StatusScheduled
}

func (t *Task) MarkRunning() {
	t.Status = StatusRunning
	t.StartedAt = time.Now()
}

func (t *Task) MarkCompleted(result any) error {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	t.Status = StatusCompleted
	t.Result = resultBytes
	t.CompletedAt = time.Now()
	return nil
}

func (t *Task) MarkFailed(errMsg string) {
	t.Status = StatusFailed
	t.Error = errMsg
	t.CompletedAt = time.Now()
}

func (t *Task) MarkCancelled() {
	t.Status = StatusCancelled
	t.CompletedAt = time.Now()
}

func (t *Task) IncrementRetry() {
	t.Retried++
	t.Status = StatusRetrying
}

func (t *Task) CanRetry() bool {
	return t.Retried < t.MaxRetries
}

func (t *Task) SetMetadata(key, value string) {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
}

func (t *Task) GetMetadata(key string) string {
	if t.Metadata == nil {
		return ""
	}
	return t.Metadata[key]
}

func (t *Task) UnmarshalPayload(v any) error {
	return json.Unmarshal(t.Payload, v)
}

func (t *Task) UnmarshalResult(v any) error {
	return json.Unmarshal(t.Result, v)
}

func (t *Task) IsTerminal() bool {
	return t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusCancelled
}

func (s Status) String() string {
	return string(s)
}
