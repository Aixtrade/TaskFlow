package task

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/domain/task"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
)

var (
	ErrInvalidTaskType = errors.New("invalid task type")
	ErrInvalidPayload  = errors.New("invalid payload")
	ErrInvalidTaskID   = errors.New("invalid task id")
	ErrInvalidQueue    = errors.New("invalid queue")
	ErrTaskNotFound    = errors.New("task not found")
)

type Service struct {
	client *asynqqueue.Client
	logger *zap.Logger
}

func NewService(client *asynqqueue.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

type CreateTaskResult struct {
	TaskID string `json:"task_id"`
	Queue  string `json:"queue"`
	Status string `json:"status"`
}

func (s *Service) CreateTask(ctx context.Context, cmd *CreateTaskCommand) (*CreateTaskResult, error) {
	if err := cmd.Validate(); err != nil {
		return nil, err
	}

	t, err := task.NewTask(cmd.Type, cmd.Payload)
	if err != nil {
		return nil, err
	}

	t.ID = uuid.New().String()

	if cmd.Queue != "" {
		t.Queue = cmd.Queue
	}
	if cmd.MaxRetries > 0 {
		t.MaxRetries = cmd.MaxRetries
	}
	if cmd.Timeout > 0 {
		t.Timeout = cmd.Timeout
	}
	if !cmd.ProcessAt.IsZero() {
		t.SetScheduledAt(cmd.ProcessAt)
	}
	for k, v := range cmd.Metadata {
		t.SetMetadata(k, v)
	}

	opts := asynqqueue.EnqueueOptions{
		Queue:      t.Queue,
		MaxRetries: t.MaxRetries,
		Timeout:    t.Timeout,
		ProcessAt:  cmd.ProcessAt,
		Unique:     cmd.Unique,
		TaskID:     t.ID,
	}

	info, err := s.client.Enqueue(ctx, t, opts)
	if err != nil {
		s.logger.Error("failed to enqueue task",
			zap.String("type", t.Type.String()),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("task created",
		zap.String("task_id", info.ID),
		zap.String("type", t.Type.String()),
		zap.String("queue", info.Queue),
	)

	return &CreateTaskResult{
		TaskID: info.ID,
		Queue:  info.Queue,
		Status: info.State.String(),
	}, nil
}

type TaskInfo struct {
	ID          string `json:"id"`
	Queue       string `json:"queue"`
	Type        string `json:"type"`
	State       string `json:"state"`
	MaxRetry    int    `json:"max_retry"`
	Retried     int    `json:"retried"`
	LastErr     string `json:"last_err,omitempty"`
	NextProcessAt string `json:"next_process_at,omitempty"`
}

func (s *Service) GetTask(ctx context.Context, query *GetTaskQuery) (*TaskInfo, error) {
	if err := query.Validate(); err != nil {
		return nil, err
	}

	info, err := s.client.GetTaskInfo(query.Queue, query.TaskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	result := &TaskInfo{
		ID:       info.ID,
		Queue:    info.Queue,
		Type:     info.Type,
		State:    info.State.String(),
		MaxRetry: info.MaxRetry,
		Retried:  info.Retried,
		LastErr:  info.LastErr,
	}

	if !info.NextProcessAt.IsZero() {
		result.NextProcessAt = info.NextProcessAt.Format("2006-01-02T15:04:05Z07:00")
	}

	return result, nil
}

func (s *Service) CancelTask(ctx context.Context, cmd *CancelTaskCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	err := s.client.CancelTask(cmd.TaskID)
	if err != nil {
		s.logger.Error("failed to cancel task",
			zap.String("task_id", cmd.TaskID),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("task cancelled", zap.String("task_id", cmd.TaskID))
	return nil
}

func (s *Service) DeleteTask(ctx context.Context, cmd *DeleteTaskCommand) error {
	if err := cmd.Validate(); err != nil {
		return err
	}

	err := s.client.DeleteTask(cmd.Queue, cmd.TaskID)
	if err != nil {
		s.logger.Error("failed to delete task",
			zap.String("task_id", cmd.TaskID),
			zap.String("queue", cmd.Queue),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("task deleted",
		zap.String("task_id", cmd.TaskID),
		zap.String("queue", cmd.Queue),
	)
	return nil
}

func (s *Service) GetQueueStats(ctx context.Context, query *GetQueueStatsQuery) ([]asynqqueue.QueueStats, error) {
	if query.Queue != "" {
		info, err := s.client.GetQueueInfo(query.Queue)
		if err != nil {
			return nil, err
		}
		return []asynqqueue.QueueStats{{
			Queue:     query.Queue,
			Pending:   info.Pending,
			Active:    info.Active,
			Scheduled: info.Scheduled,
			Retry:     info.Retry,
			Archived:  info.Archived,
			Completed: info.Completed,
		}}, nil
	}

	return s.client.GetAllQueueStats()
}
