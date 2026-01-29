package worker

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

func LoggingMiddleware(logger *zap.Logger) asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			start := time.Now()
			taskID := GetTaskID(ctx)

			logger.Info("processing task",
				zap.String("type", t.Type()),
				zap.String("task_id", taskID),
				zap.Int("retry", GetRetryCount(ctx)),
			)

			err := h.ProcessTask(ctx, t)

			duration := time.Since(start)

			if err != nil {
				logger.Error("task failed",
					zap.String("type", t.Type()),
					zap.String("task_id", taskID),
					zap.Duration("duration", duration),
					zap.Error(err),
				)
			} else {
				logger.Info("task completed",
					zap.String("type", t.Type()),
					zap.String("task_id", taskID),
					zap.Duration("duration", duration),
				)
			}

			return err
		})
	}
}

func RecoveryMiddleware(logger *zap.Logger) asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("task panic recovered",
						zap.String("type", t.Type()),
						zap.String("task_id", GetTaskID(ctx)),
						zap.Any("panic", r),
					)
					err = asynq.SkipRetry
				}
			}()

			return h.ProcessTask(ctx, t)
		})
	}
}

func TimeoutMiddleware(timeout time.Duration) asynq.MiddlewareFunc {
	return func(h asynq.Handler) asynq.Handler {
		return asynq.HandlerFunc(func(ctx context.Context, t *asynq.Task) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan error, 1)
			go func() {
				done <- h.ProcessTask(ctx, t)
			}()

			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
}
