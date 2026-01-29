package task

import apperrors "github.com/Aixtrade/TaskFlow/pkg/errors"

type GetTaskQuery struct {
	TaskID string `json:"task_id"`
	Queue  string `json:"queue"`
}

func (q *GetTaskQuery) Validate() error {
	if q.TaskID == "" {
		return apperrors.ErrInvalidTaskID
	}
	if q.Queue == "" {
		return apperrors.ErrInvalidQueue
	}
	return nil
}

type GetQueueStatsQuery struct {
	Queue string `json:"queue,omitempty"`
}

type ListTasksQuery struct {
	Queue  string `json:"queue"`
	Status string `json:"status"`
	Page   int    `json:"page"`
	Size   int    `json:"size"`
}

func (q *ListTasksQuery) Validate() error {
	if q.Queue == "" {
		return apperrors.ErrInvalidQueue
	}
	if q.Status == "" {
		q.Status = "active"
	}
	switch q.Status {
	case "pending", "active", "scheduled", "retry", "archived", "completed":
	default:
		return apperrors.ErrInvalidTaskState
	}
	if q.Page < 0 {
		q.Page = 0
	}
	if q.Size <= 0 {
		q.Size = 20
	}
	return nil
}
