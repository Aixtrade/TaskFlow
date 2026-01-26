package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Publisher 进度发布器
type Publisher struct {
	redis   *redis.Client
	logger  *zap.Logger
	options StreamOptions
}

// NewPublisher 创建进度发布器
func NewPublisher(redisClient *redis.Client, logger *zap.Logger, opts ...StreamOptions) *Publisher {
	opt := DefaultOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	return &Publisher{
		redis:   redisClient,
		logger:  logger,
		options: opt,
	}
}

// Publish 发布进度到 Redis Stream
func (p *Publisher) Publish(ctx context.Context, prog *Progress) error {
	if prog == nil {
		return fmt.Errorf("progress cannot be nil")
	}

	key := StreamKey(prog.TaskID)

	// 构建 Stream 数据
	values := map[string]interface{}{
		"task_id":      prog.TaskID,
		"percentage":   prog.Percentage,
		"stage":        prog.Stage,
		"message":      prog.Message,
		"timestamp_ms": prog.TimestampMs,
	}

	// 添加 metadata（如果有）
	if len(prog.Metadata) > 0 {
		metaJSON, err := json.Marshal(prog.Metadata)
		if err == nil {
			values["metadata"] = string(metaJSON)
		}
	}

	// 发布到 Stream（XADD）
	args := &redis.XAddArgs{
		Stream: key,
		Values: values,
	}

	// 限制 Stream 长度
	if p.options.MaxLen > 0 {
		args.MaxLen = p.options.MaxLen
		args.Approx = true // 使用 ~ 近似限制，性能更好
	}

	result, err := p.redis.XAdd(ctx, args).Result()
	if err != nil {
		p.logger.Error("failed to publish progress",
			zap.String("task_id", prog.TaskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish progress: %w", err)
	}

	// 设置 TTL（如果是第一条消息）
	p.ensureTTL(ctx, key)

	p.logger.Debug("progress published",
		zap.String("task_id", prog.TaskID),
		zap.String("stream_id", result),
		zap.Int32("percentage", prog.Percentage),
	)

	return nil
}

// PublishCompletion 发布任务完成事件
func (p *Publisher) PublishCompletion(ctx context.Context, taskID, status, message string) error {
	key := StreamKey(taskID)

	// 发布完成消息到同一个 Stream
	values := map[string]interface{}{
		"task_id":      taskID,
		"percentage":   100,
		"stage":        "completed",
		"message":      message,
		"status":       status, // completed, failed, cancelled
		"timestamp_ms": time.Now().UnixMilli(),
		"is_final":     "true", // 标记为最终消息
	}

	args := &redis.XAddArgs{
		Stream: key,
		Values: values,
	}

	if p.options.MaxLen > 0 {
		args.MaxLen = p.options.MaxLen
		args.Approx = true
	}

	_, err := p.redis.XAdd(ctx, args).Result()
	if err != nil {
		p.logger.Error("failed to publish completion",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish completion: %w", err)
	}

	p.logger.Debug("completion published",
		zap.String("task_id", taskID),
		zap.String("status", status),
	)

	return nil
}

// ensureTTL 确保 Stream 设置了过期时间
func (p *Publisher) ensureTTL(ctx context.Context, key string) {
	if p.options.TTL <= 0 {
		return
	}

	// 检查是否已设置 TTL
	ttl, err := p.redis.TTL(ctx, key).Result()
	if err != nil {
		return
	}

	// 如果没有设置 TTL，则设置
	if ttl < 0 {
		p.redis.Expire(ctx, key, p.options.TTL)
	}
}

// Delete 删除任务的进度 Stream
func (p *Publisher) Delete(ctx context.Context, taskID string) error {
	key := StreamKey(taskID)
	return p.redis.Del(ctx, key).Err()
}

// Exists 检查任务进度 Stream 是否存在
func (p *Publisher) Exists(ctx context.Context, taskID string) (bool, error) {
	key := StreamKey(taskID)
	n, err := p.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
