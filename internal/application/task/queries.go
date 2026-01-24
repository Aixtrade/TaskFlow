package task

type GetTaskQuery struct {
	TaskID string `json:"task_id"`
	Queue  string `json:"queue"`
}

func (q *GetTaskQuery) Validate() error {
	if q.TaskID == "" {
		return ErrInvalidTaskID
	}
	if q.Queue == "" {
		return ErrInvalidQueue
	}
	return nil
}

type ListTasksQuery struct {
	Queue   string   `json:"queue,omitempty"`
	Status  []string `json:"status,omitempty"`
	Type    []string `json:"type,omitempty"`
	Offset  int      `json:"offset,omitempty"`
	Limit   int      `json:"limit,omitempty"`
}

func (q *ListTasksQuery) Validate() error {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	return nil
}

type GetQueueStatsQuery struct {
	Queue string `json:"queue,omitempty"`
}
