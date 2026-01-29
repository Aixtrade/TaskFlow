package grpc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	pb "github.com/Aixtrade/TaskFlow/api/proto/grpc_task/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/protobuf/types/known/structpb"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	Address             string        `mapstructure:"address"`
	Timeout             time.Duration `mapstructure:"timeout"`
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	MaxRetries          int           `mapstructure:"max_retries"`
	RetryDelay          time.Duration `mapstructure:"retry_delay"`
}

// DefaultClientConfig 返回默认配置
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:             300 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryDelay:          time.Second,
	}
}

// StreamingGRPCClient 封装与 gRPC 服务的流式通信
type StreamingGRPCClient struct {
	config  ClientConfig
	conn    *grpc.ClientConn
	client  pb.TaskExecutorServiceClient
	logger  *zap.Logger
	healthy atomic.Bool

	mu         sync.RWMutex
	cancelFunc context.CancelFunc
}

// NewStreamingGRPCClient 创建新的 gRPC 服务客户端
func NewStreamingGRPCClient(config ClientConfig, logger *zap.Logger) (*StreamingGRPCClient, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// 应用默认值
	if config.Timeout == 0 {
		config.Timeout = DefaultClientConfig().Timeout
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = DefaultClientConfig().HealthCheckInterval
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultClientConfig().MaxRetries
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = DefaultClientConfig().RetryDelay
	}

	c := &StreamingGRPCClient{
		config: config,
		logger: logger,
	}

	if err := c.connect(); err != nil {
		return nil, err
	}

	// 启动健康检查
	ctx, cancel := context.WithCancel(context.Background())
	c.cancelFunc = cancel
	go c.healthCheckLoop(ctx)

	return c, nil
}

// connect 建立 gRPC 连接
func (c *StreamingGRPCClient) connect() error {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithChainUnaryInterceptor(
			LoggingUnaryInterceptor(c.logger),
			RetryUnaryInterceptor(c.config.MaxRetries, c.config.RetryDelay, c.logger),
			MetadataUnaryInterceptor("taskflow-worker"),
		),
		grpc.WithChainStreamInterceptor(
			LoggingStreamInterceptor(c.logger),
			MetadataStreamInterceptor("taskflow-worker"),
		),
	}

	conn, err := grpc.NewClient(c.config.Address, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.config.Address, err)
	}

	c.conn = conn
	c.client = pb.NewTaskExecutorServiceClient(conn)
	c.healthy.Store(true)

	c.logger.Info("connected to grpc service",
		zap.String("address", c.config.Address),
	)

	return nil
}

// healthCheckLoop 定期执行健康检查
func (c *StreamingGRPCClient) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(c.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkHealth(ctx)
		}
	}
}

// checkHealth 执行单次健康检查
func (c *StreamingGRPCClient) checkHealth(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.client.HealthCheck(checkCtx, &pb.HealthCheckRequest{})
	if err != nil {
		c.logger.Warn("health check failed",
			zap.String("address", c.config.Address),
			zap.Error(err),
		)
		c.healthy.Store(false)
		return
	}

	healthy := resp.Status == pb.HealthStatus_HEALTH_STATUS_HEALTHY
	c.healthy.Store(healthy)

	if !healthy {
		c.logger.Warn("service unhealthy",
			zap.String("address", c.config.Address),
			zap.String("status", resp.Status.String()),
			zap.String("message", resp.Message),
		)
	}
}

// IsHealthy 返回服务健康状态
func (c *StreamingGRPCClient) IsHealthy() bool {
	// 同时检查连接状态
	if c.conn != nil && c.conn.GetState() == connectivity.TransientFailure {
		return false
	}
	return c.healthy.Load()
}

// ProgressCallback 进度回调函数类型
type ProgressCallback func(*pb.Progress)

// ExecuteTask 执行任务并返回结果
func (c *StreamingGRPCClient) ExecuteTask(
	ctx context.Context,
	req *pb.ExecuteTaskRequest,
	onProgress ProgressCallback,
) (*pb.TaskResult, error) {
	// 设置超时
	timeout := c.config.Timeout
	if req.Options != nil && req.Options.TimeoutMs > 0 {
		timeout = time.Duration(req.Options.TimeoutMs) * time.Millisecond
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 发起流式调用
	stream, err := c.client.ExecuteTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start task execution: %w", err)
	}

	// 处理流式响应
	var result *pb.TaskResult
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("stream error: %w", err)
		}

		switch r := resp.Response.(type) {
		case *pb.ExecuteTaskResponse_Progress:
			if onProgress != nil {
				onProgress(r.Progress)
			}
		case *pb.ExecuteTaskResponse_Result:
			result = r.Result
		case *pb.ExecuteTaskResponse_Error:
			return nil, &GRPCError{
				Code:      r.Error.Code,
				Message:   r.Error.Message,
				Retryable: r.Error.Retryable,
			}
		}
	}

	if result == nil {
		return nil, fmt.Errorf("no result received from stream")
	}

	return result, nil
}

// CancelTask 取消任务
func (c *StreamingGRPCClient) CancelTask(ctx context.Context, taskID, reason string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := c.client.CancelTask(ctx, &pb.CancelTaskRequest{
		TaskId: taskID,
		Reason: reason,
	})
	if err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("cancel failed: %s", resp.Message)
	}

	return nil
}

// Close 关闭客户端连接
func (c *StreamingGRPCClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	c.logger.Info("closed grpc service client",
		zap.String("address", c.config.Address),
	)

	return nil
}

// Address 返回服务地址
func (c *StreamingGRPCClient) Address() string {
	return c.config.Address
}

// BuildPayloadStruct 将 map 转换为 protobuf Struct
func BuildPayloadStruct(data map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(data)
}
