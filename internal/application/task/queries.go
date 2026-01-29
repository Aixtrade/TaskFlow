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
