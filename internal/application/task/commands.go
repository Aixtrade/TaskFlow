package task

import (
	"encoding/json"
	"time"

	apperrors "github.com/Aixtrade/TaskFlow/pkg/errors"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type CreateTaskCommand struct {
	Type       tasktype.Type     `json:"type"`
	Payload    json.RawMessage   `json:"payload"`
	Queue      string            `json:"queue,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	ProcessAt  time.Time         `json:"process_at,omitempty"`
	Unique     time.Duration     `json:"unique,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (c *CreateTaskCommand) Validate() error {
	if !c.Type.IsValid() {
		return apperrors.ErrInvalidTaskType
	}
	if len(c.Payload) == 0 {
		return apperrors.ErrInvalidPayload
	}
	return nil
}

type CancelTaskCommand struct {
	TaskID string `json:"task_id"`
}

func (c *CancelTaskCommand) Validate() error {
	if c.TaskID == "" {
		return apperrors.ErrInvalidTaskID
	}
	return nil
}

type DeleteTaskCommand struct {
	TaskID string `json:"task_id"`
	Queue  string `json:"queue"`
}

func (c *DeleteTaskCommand) Validate() error {
	if c.TaskID == "" {
		return apperrors.ErrInvalidTaskID
	}
	if c.Queue == "" {
		return apperrors.ErrInvalidQueue
	}
	return nil
}
