package worker

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type Handler interface {
	ProcessTask(ctx context.Context, task *asynq.Task) error
	Type() string
}

type BaseHandler struct {
	logger *zap.Logger
}

func NewBaseHandler(logger *zap.Logger) *BaseHandler {
	return &BaseHandler{
		logger: logger,
	}
}

func (h *BaseHandler) Logger() *zap.Logger {
	return h.logger
}

func (h *BaseHandler) LogTaskStart(taskType, taskID string) {
	h.logger.Info("task started",
		zap.String("type", taskType),
		zap.String("task_id", taskID),
	)
}

func (h *BaseHandler) LogTaskComplete(taskType, taskID string) {
	h.logger.Info("task completed",
		zap.String("type", taskType),
		zap.String("task_id", taskID),
	)
}

func (h *BaseHandler) LogTaskError(taskType, taskID string, err error) {
	h.logger.Error("task failed",
		zap.String("type", taskType),
		zap.String("task_id", taskID),
		zap.Error(err),
	)
}

func UnmarshalPayload[T any](task *asynq.Task) (*T, error) {
	var payload T
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func GetTaskID(ctx context.Context) string {
	id, ok := asynq.GetTaskID(ctx)
	if !ok {
		return ""
	}
	return id
}

func GetRetryCount(ctx context.Context) int {
	count, ok := asynq.GetRetryCount(ctx)
	if !ok {
		return 0
	}
	return count
}

func GetMaxRetry(ctx context.Context) int {
	max, ok := asynq.GetMaxRetry(ctx)
	if !ok {
		return 0
	}
	return max
}

func GetQueueName(ctx context.Context) string {
	queue, ok := asynq.GetQueueName(ctx)
	if !ok {
		return ""
	}
	return queue
}
