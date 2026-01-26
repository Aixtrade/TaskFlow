package progress

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Subscriber 进度订阅器
type Subscriber struct {
	redis   *redis.Client
	logger  *zap.Logger
	options StreamOptions
}

// NewSubscriber 创建进度订阅器
func NewSubscriber(redisClient *redis.Client, logger *zap.Logger, opts ...StreamOptions) *Subscriber {
	opt := DefaultOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	return &Subscriber{
		redis:   redisClient,
		logger:  logger,
		options: opt,
	}
}

// SubscribeResult 订阅结果
type SubscribeResult struct {
	Progress  *Progress // 进度数据
	IsFinal   bool      // 是否是最终消息
	Status    string    // 最终状态（仅当 IsFinal 为 true）
	StreamID  string    // Redis Stream ID
	Error     error     // 错误信息
}

// Subscribe 订阅任务进度
// 返回一个 channel，持续接收进度更新直到任务完成或 context 取消
func (s *Subscriber) Subscribe(ctx context.Context, taskID string, startID ...string) <-chan SubscribeResult {
	ch := make(chan SubscribeResult, 10)

	// 默认从最新消息开始读取，使用 $ 表示只读新消息
	// 如果指定了 startID，则从该位置开始读取
	lastID := "$"
	if len(startID) > 0 && startID[0] != "" {
		lastID = startID[0]
	}

	go func() {
		defer close(ch)

		key := StreamKey(taskID)
		blockTimeout := s.options.ReadTimeout
		if blockTimeout == 0 {
			blockTimeout = 30 * time.Second
		}

		for {
			select {
			case <-ctx.Done():
				s.logger.Debug("subscription cancelled",
					zap.String("task_id", taskID),
					zap.Error(ctx.Err()),
				)
				return
			default:
			}

			// 使用 XREAD 阻塞读取
			streams, err := s.redis.XRead(ctx, &redis.XReadArgs{
				Streams: []string{key, lastID},
				Block:   blockTimeout,
				Count:   10, // 每次最多读取 10 条
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// 超时，继续等待
					continue
				}
				if ctx.Err() != nil {
					// context 已取消
					return
				}
				s.logger.Error("failed to read stream",
					zap.String("task_id", taskID),
					zap.Error(err),
				)
				ch <- SubscribeResult{Error: err}
				return
			}

			// 处理读取到的消息
			for _, stream := range streams {
				for _, msg := range stream.Messages {
					result := s.parseMessage(taskID, msg)
					lastID = msg.ID

					select {
					case ch <- result:
					case <-ctx.Done():
						return
					}

					// 如果是最终消息，结束订阅
					if result.IsFinal {
						s.logger.Debug("received final message, closing subscription",
							zap.String("task_id", taskID),
							zap.String("status", result.Status),
						)
						return
					}
				}
			}
		}
	}()

	return ch
}

// GetHistory 获取任务的历史进度
// startID: 起始 ID（"-" 表示从头开始）
// count: 获取数量（0 表示全部）
func (s *Subscriber) GetHistory(ctx context.Context, taskID string, startID string, count int64) ([]SubscribeResult, error) {
	key := StreamKey(taskID)

	if startID == "" {
		startID = "-"
	}

	var messages []redis.XMessage
	var err error

	if count > 0 {
		messages, err = s.redis.XRangeN(ctx, key, startID, "+", count).Result()
	} else {
		messages, err = s.redis.XRange(ctx, key, startID, "+").Result()
	}

	if err != nil {
		return nil, err
	}

	results := make([]SubscribeResult, 0, len(messages))
	for _, msg := range messages {
		results = append(results, s.parseMessage(taskID, msg))
	}

	return results, nil
}

// GetLatest 获取最新的进度
func (s *Subscriber) GetLatest(ctx context.Context, taskID string) (*SubscribeResult, error) {
	key := StreamKey(taskID)

	// 使用 XREVRANGE 获取最后一条消息
	messages, err := s.redis.XRevRangeN(ctx, key, "+", "-", 1).Result()
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		return nil, nil
	}

	result := s.parseMessage(taskID, messages[0])
	return &result, nil
}

// parseMessage 解析 Stream 消息
func (s *Subscriber) parseMessage(taskID string, msg redis.XMessage) SubscribeResult {
	result := SubscribeResult{
		StreamID: msg.ID,
		Progress: &Progress{
			TaskID: taskID,
		},
	}

	values := msg.Values

	// 解析 percentage
	if v, ok := values["percentage"]; ok {
		switch val := v.(type) {
		case string:
			if p, err := strconv.ParseInt(val, 10, 32); err == nil {
				result.Progress.Percentage = int32(p)
			}
		case int64:
			result.Progress.Percentage = int32(val)
		}
	}

	// 解析 stage
	if v, ok := values["stage"].(string); ok {
		result.Progress.Stage = v
	}

	// 解析 message
	if v, ok := values["message"].(string); ok {
		result.Progress.Message = v
	}

	// 解析 timestamp_ms
	if v, ok := values["timestamp_ms"]; ok {
		switch val := v.(type) {
		case string:
			if ts, err := strconv.ParseInt(val, 10, 64); err == nil {
				result.Progress.TimestampMs = ts
			}
		case int64:
			result.Progress.TimestampMs = val
		}
	}

	// 解析 metadata
	if v, ok := values["metadata"].(string); ok && v != "" {
		var meta map[string]string
		if err := json.Unmarshal([]byte(v), &meta); err == nil {
			result.Progress.Metadata = meta
		}
	}

	// 检查是否是最终消息
	if v, ok := values["is_final"].(string); ok && v == "true" {
		result.IsFinal = true
		if status, ok := values["status"].(string); ok {
			result.Status = status
		}
	}

	return result
}

// StreamInfo 获取 Stream 信息
type StreamInfo struct {
	Length      int64  // Stream 长度
	FirstEntry  string // 第一条消息 ID
	LastEntry   string // 最后一条消息 ID
	HasProgress bool   // 是否有进度数据
}

// GetStreamInfo 获取任务进度 Stream 的信息
func (s *Subscriber) GetStreamInfo(ctx context.Context, taskID string) (*StreamInfo, error) {
	key := StreamKey(taskID)

	// 检查 key 是否存在
	exists, err := s.redis.Exists(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if exists == 0 {
		return &StreamInfo{HasProgress: false}, nil
	}

	// 获取 Stream 长度
	length, err := s.redis.XLen(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	info := &StreamInfo{
		Length:      length,
		HasProgress: length > 0,
	}

	// 获取第一条和最后一条消息 ID
	if length > 0 {
		// 第一条
		first, err := s.redis.XRangeN(ctx, key, "-", "+", 1).Result()
		if err == nil && len(first) > 0 {
			info.FirstEntry = first[0].ID
		}

		// 最后一条
		last, err := s.redis.XRevRangeN(ctx, key, "+", "-", 1).Result()
		if err == nil && len(last) > 0 {
			info.LastEntry = last[0].ID
		}
	}

	return info, nil
}
