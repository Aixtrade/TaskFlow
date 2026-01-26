package grpctask

import (
	"context"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	grpcclient "github.com/Aixtrade/TaskFlow/internal/infrastructure/grpc"
	"github.com/Aixtrade/TaskFlow/internal/worker"
	"github.com/Aixtrade/TaskFlow/pkg/payload"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"

	pb "github.com/Aixtrade/TaskFlow/api/proto/grpc_task/v1"
)

// Config 包含 gRPC 服务的配置
type Config struct {
	Services map[string]grpcclient.ClientConfig `mapstructure:"services"`
	Defaults grpcclient.ClientConfig            `mapstructure:"defaults"`
}

// Handler 处理所有 gRPC 任务
type Handler struct {
	*worker.BaseHandler
	clientManager *grpcclient.ClientManager
	config        Config
}

// NewHandler 创建新的 gRPC handler
func NewHandler(logger *zap.Logger, clientManager *grpcclient.ClientManager, cfg Config) *Handler {
	return &Handler{
		BaseHandler:   worker.NewBaseHandler(logger),
		clientManager: clientManager,
		config:        cfg,
	}
}

// Type 返回任务类型标识
func (h *Handler) Type() string {
	return tasktype.GRPCTask.String()
}

// ProcessTask 处理 gRPC 任务
func (h *Handler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	taskID := worker.GetTaskID(ctx)
	h.LogTaskStart(h.Type(), taskID)

	// 1. 解析 payload
	p, err := worker.UnmarshalPayload[payload.GRPCTaskPayload](task)
	if err != nil {
		h.Logger().Error("failed to unmarshal payload",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return asynq.SkipRetry // payload 格式错误，不重试
	}

	// 2. 验证 payload
	if err := p.Validate(); err != nil {
		h.Logger().Error("invalid payload",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return asynq.SkipRetry
	}

	// 3. 验证服务是否存在
	if !h.clientManager.HasService(p.Service) {
		h.Logger().Error("unknown service",
			zap.String("task_id", taskID),
			zap.String("service", p.Service),
		)
		return asynq.SkipRetry // 未知服务，不重试
	}

	// 4. 获取客户端
	client, err := h.clientManager.GetClient(p.Service)
	if err != nil {
		h.Logger().Error("failed to get client",
			zap.String("task_id", taskID),
			zap.String("service", p.Service),
			zap.Error(err),
		)
		return fmt.Errorf("failed to get client for %s: %w", p.Service, err)
	}

	// 5. 检查健康状态
	if !client.IsHealthy() {
		h.Logger().Warn("service unhealthy, will retry",
			zap.String("task_id", taskID),
			zap.String("service", p.Service),
		)
		return fmt.Errorf("service %s unavailable", p.Service) // 触发重试
	}

	// 6. 构建请求
	req, err := h.buildRequest(ctx, taskID, p)
	if err != nil {
		h.Logger().Error("failed to build request",
			zap.String("task_id", taskID),
			zap.Error(err),
		)
		return asynq.SkipRetry
	}

	// 7. 执行任务
	result, err := client.ExecuteTask(ctx, req, func(prog *pb.Progress) {
		h.Logger().Info("task progress",
			zap.String("task_id", taskID),
			zap.String("service", p.Service),
			zap.Int32("percentage", prog.Percentage),
			zap.String("stage", prog.Stage),
			zap.String("message", prog.Message),
		)
	})

	if err != nil {
		return h.handleError(taskID, p.Service, err)
	}

	// 8. 处理结果
	h.Logger().Info("task result received",
		zap.String("task_id", taskID),
		zap.String("service", p.Service),
		zap.String("status", result.Status.String()),
		zap.Int64("duration_ms", result.DurationMs),
	)

	if result.Status == pb.TaskStatus_TASK_STATUS_FAILED {
		return fmt.Errorf("task failed on grpc service")
	}

	if result.Status == pb.TaskStatus_TASK_STATUS_CANCELLED {
		return fmt.Errorf("task cancelled on grpc service")
	}

	h.LogTaskComplete(h.Type(), taskID)
	return nil
}

// buildRequest 构建 gRPC 请求
func (h *Handler) buildRequest(ctx context.Context, taskID string, p *payload.GRPCTaskPayload) (*pb.ExecuteTaskRequest, error) {
	// 获取服务配置
	serviceCfg, _ := h.clientManager.GetServiceConfig(p.Service)

	// 计算超时
	timeout := serviceCfg.Timeout
	if timeout == 0 {
		timeout = h.config.Defaults.Timeout
	}
	if timeout == 0 {
		timeout = 300 * time.Second // 默认 5 分钟
	}
	if p.Options != nil && p.Options.TimeoutMs != nil {
		timeout = time.Duration(*p.Options.TimeoutMs) * time.Millisecond
	}

	// 构建 payload struct
	dataStruct, err := grpcclient.BuildPayloadStruct(p.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload struct: %w", err)
	}

	// 构建执行选项
	enableProgress := true
	progressInterval := int32(1000)
	if p.Options != nil {
		if p.Options.EnableProgress != nil {
			enableProgress = *p.Options.EnableProgress
		}
		if p.Options.ProgressIntervalMs != nil {
			progressInterval = int32(*p.Options.ProgressIntervalMs)
		}
	}

	req := &pb.ExecuteTaskRequest{
		TaskId:   taskID,
		TaskType: p.Method,
		Payload:  dataStruct,
		Metadata: map[string]string{
			"service":     p.Service,
			"queue":       worker.GetQueueName(ctx),
			"retry_count": fmt.Sprintf("%d", worker.GetRetryCount(ctx)),
			"max_retry":   fmt.Sprintf("%d", worker.GetMaxRetry(ctx)),
		},
		Options: &pb.ExecutionOptions{
			TimeoutMs:          int64(timeout.Milliseconds()),
			EnableProgress:     enableProgress,
			ProgressIntervalMs: progressInterval,
		},
	}

	return req, nil
}

// handleError 处理执行错误
func (h *Handler) handleError(taskID, service string, err error) error {
	grpcErr, ok := grpcclient.ConvertError(err)
	if ok {
		h.Logger().Error("grpc service error",
			zap.String("task_id", taskID),
			zap.String("service", service),
			zap.String("code", grpcErr.Code),
			zap.String("message", grpcErr.Message),
			zap.Bool("retryable", grpcErr.Retryable),
		)
		if !grpcErr.Retryable {
			return asynq.SkipRetry
		}
	} else {
		h.LogTaskError(h.Type(), taskID, err)
	}
	return err
}
