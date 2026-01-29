package dto

import (
	"encoding/json"
	"time"

	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type CreateTaskRequest struct {
	Type       string            `json:"type" binding:"required"`
	Payload    json.RawMessage   `json:"payload" binding:"required"`
	Queue      string            `json:"queue,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Timeout    string            `json:"timeout,omitempty"`
	ProcessAt  string            `json:"process_at,omitempty"`
	Unique     string            `json:"unique,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (r *CreateTaskRequest) GetTimeout() (time.Duration, error) {
	if r.Timeout == "" {
		return 0, nil
	}
	return time.ParseDuration(r.Timeout)
}

func (r *CreateTaskRequest) GetProcessAt() (time.Time, error) {
	if r.ProcessAt == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, r.ProcessAt)
}

func (r *CreateTaskRequest) GetUnique() (time.Duration, error) {
	if r.Unique == "" {
		return 0, nil
	}
	return time.ParseDuration(r.Unique)
}

func (r *CreateTaskRequest) GetTaskType() tasktype.Type {
	return tasktype.Type(r.Type)
}

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
	Queue  string `json:"queue"`
	Status string `json:"status"`
}

type GetTaskResponse struct {
	ID            string `json:"id"`
	Queue         string `json:"queue"`
	Type          string `json:"type"`
	State         string `json:"state"`
	MaxRetry      int    `json:"max_retry"`
	Retried       int    `json:"retried"`
	LastErr       string `json:"last_err,omitempty"`
	NextProcessAt string `json:"next_process_at,omitempty"`
}

type TaskListResponse struct {
	ID    string `json:"id"`
	Queue string `json:"queue"`
	Type  string `json:"type"`
	State string `json:"state"`
}

type QueueStatsResponse struct {
	Queue     string `json:"queue"`
	Pending   int    `json:"pending"`
	Active    int    `json:"active"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}
