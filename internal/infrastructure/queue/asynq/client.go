package asynq

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hibiken/asynq"

	"github.com/Aixtrade/TaskFlow/internal/config"
	"github.com/Aixtrade/TaskFlow/internal/domain/task"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Client struct {
	client    *asynq.Client
	inspector *asynq.Inspector
}

func NewClient(cfg *config.RedisConfig) (*Client, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	return &Client{
		client:    client,
		inspector: inspector,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

type EnqueueOptions struct {
	Queue      string
	MaxRetries int
	Timeout    time.Duration
	Deadline   time.Time
	ProcessAt  time.Time
	Unique     time.Duration
	TaskID     string
}

func DefaultEnqueueOptions() EnqueueOptions {
	return EnqueueOptions{
		Queue:      "default",
		MaxRetries: 3,
		Timeout:    30 * time.Minute,
	}
}

func (c *Client) Enqueue(ctx context.Context, t *task.Task, opts ...EnqueueOptions) (*asynq.TaskInfo, error) {
	opt := DefaultEnqueueOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	if t.Queue != "" {
		opt.Queue = t.Queue
	}
	if t.MaxRetries > 0 {
		opt.MaxRetries = t.MaxRetries
	}
	if t.Timeout > 0 {
		opt.Timeout = t.Timeout
	}

	asynqOpts := []asynq.Option{
		asynq.Queue(opt.Queue),
		asynq.MaxRetry(opt.MaxRetries),
		asynq.Timeout(opt.Timeout),
	}

	if !opt.Deadline.IsZero() {
		asynqOpts = append(asynqOpts, asynq.Deadline(opt.Deadline))
	}

	if !opt.ProcessAt.IsZero() {
		asynqOpts = append(asynqOpts, asynq.ProcessAt(opt.ProcessAt))
	}

	if opt.Unique > 0 {
		asynqOpts = append(asynqOpts, asynq.Unique(opt.Unique))
	}

	if opt.TaskID != "" {
		asynqOpts = append(asynqOpts, asynq.TaskID(opt.TaskID))
	} else if t.ID != "" {
		asynqOpts = append(asynqOpts, asynq.TaskID(t.ID))
	}

	asynqTask := asynq.NewTask(t.Type.String(), t.Payload)

	return c.client.EnqueueContext(ctx, asynqTask, asynqOpts...)
}

func (c *Client) EnqueueTask(ctx context.Context, taskType tasktype.Type, payload any, opts ...EnqueueOptions) (*asynq.TaskInfo, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	opt := DefaultEnqueueOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}

	if opt.Queue == "default" {
		opt.Queue = taskType.Queue()
	}

	asynqOpts := []asynq.Option{
		asynq.Queue(opt.Queue),
		asynq.MaxRetry(opt.MaxRetries),
		asynq.Timeout(opt.Timeout),
	}

	if !opt.Deadline.IsZero() {
		asynqOpts = append(asynqOpts, asynq.Deadline(opt.Deadline))
	}

	if !opt.ProcessAt.IsZero() {
		asynqOpts = append(asynqOpts, asynq.ProcessAt(opt.ProcessAt))
	}

	if opt.Unique > 0 {
		asynqOpts = append(asynqOpts, asynq.Unique(opt.Unique))
	}

	if opt.TaskID != "" {
		asynqOpts = append(asynqOpts, asynq.TaskID(opt.TaskID))
	}

	asynqTask := asynq.NewTask(taskType.String(), payloadBytes)

	return c.client.EnqueueContext(ctx, asynqTask, asynqOpts...)
}

func (c *Client) CancelTask(taskID string) error {
	return c.inspector.CancelProcessing(taskID)
}

func (c *Client) DeleteTask(queue, taskID string) error {
	return c.inspector.DeleteTask(queue, taskID)
}

func (c *Client) GetTaskInfo(queue, taskID string) (*asynq.TaskInfo, error) {
	return c.inspector.GetTaskInfo(queue, taskID)
}

func (c *Client) ListActiveTasks(queue string, page, size int) ([]*asynq.TaskInfo, error) {
	return c.inspector.ListActiveTasks(queue, page, size)
}

func (c *Client) ListTasks(queue, state string, page, size int) ([]*asynq.TaskInfo, error) {
	switch state {
	case "active":
		return c.inspector.ListActiveTasks(queue, page, size)
	case "pending":
		return c.inspector.ListPendingTasks(queue, page, size)
	case "scheduled":
		return c.inspector.ListScheduledTasks(queue, page, size)
	case "retry":
		return c.inspector.ListRetryTasks(queue, page, size)
	case "archived":
		return c.inspector.ListArchivedTasks(queue, page, size)
	case "completed":
		return c.inspector.ListCompletedTasks(queue, page, size)
	default:
		return nil, errors.New("invalid task state")
	}
}

func (c *Client) GetQueueInfo(queue string) (*asynq.QueueInfo, error) {
	return c.inspector.GetQueueInfo(queue)
}

func (c *Client) GetQueues() ([]string, error) {
	return c.inspector.Queues()
}

type QueueStats struct {
	Queue     string `json:"queue"`
	Pending   int    `json:"pending"`
	Active    int    `json:"active"`
	Scheduled int    `json:"scheduled"`
	Retry     int    `json:"retry"`
	Archived  int    `json:"archived"`
	Completed int    `json:"completed"`
}

func (c *Client) GetAllQueueStats() ([]QueueStats, error) {
	queues, err := c.inspector.Queues()
	if err != nil {
		return nil, err
	}

	var stats []QueueStats
	for _, q := range queues {
		info, err := c.inspector.GetQueueInfo(q)
		if err != nil {
			continue
		}

		stats = append(stats, QueueStats{
			Queue:     q,
			Pending:   info.Pending,
			Active:    info.Active,
			Scheduled: info.Scheduled,
			Retry:     info.Retry,
			Archived:  info.Archived,
			Completed: info.Completed,
		})
	}

	return stats, nil
}

func (c *Client) PauseQueue(queue string) error {
	return c.inspector.PauseQueue(queue)
}

func (c *Client) UnpauseQueue(queue string) error {
	return c.inspector.UnpauseQueue(queue)
}
