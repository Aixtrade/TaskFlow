package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/worker"
	"github.com/Aixtrade/TaskFlow/pkg/payload"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Handler struct {
	*worker.BaseHandler
}

func NewHandler(logger *zap.Logger) *Handler {
	return &Handler{
		BaseHandler: worker.NewBaseHandler(logger),
	}
}

func (h *Handler) Type() string {
	return tasktype.Demo.String()
}

func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	taskID := worker.GetTaskID(ctx)
	h.LogTaskStart(h.Type(), taskID)

	p, err := worker.UnmarshalPayload[payload.DemoPayload](task)
	if err != nil {
		h.LogTaskError(h.Type(), taskID, err)
		return err
	}

	h.Logger().Info("========== Demo Task Started ==========")
	h.Logger().Info(fmt.Sprintf("Task ID: %s", taskID))
	h.Logger().Info(fmt.Sprintf("Message: %s", p.Message))
	h.Logger().Info(fmt.Sprintf("Count: %d", p.Count))
	h.Logger().Info(fmt.Sprintf("Queue: %s", worker.GetQueueName(ctx)))
	h.Logger().Info(fmt.Sprintf("Retry: %d / %d", worker.GetRetryCount(ctx), worker.GetMaxRetry(ctx)))

	// 模拟任务处理
	for i := 1; i <= p.Count; i++ {
		select {
		case <-ctx.Done():
			h.Logger().Warn("task cancelled")
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			h.Logger().Info(fmt.Sprintf("Processing step %d/%d...", i, p.Count))
		}
	}

	h.Logger().Info("========== Demo Task Completed ==========")
	h.LogTaskComplete(h.Type(), taskID)

	return nil
}
