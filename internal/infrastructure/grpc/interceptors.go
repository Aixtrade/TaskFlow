package grpc

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// LoggingUnaryInterceptor 创建一元 RPC 日志拦截器
func LoggingUnaryInterceptor(logger *zap.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()

		logger.Debug("grpc call started",
			zap.String("method", method),
			zap.String("target", cc.Target()),
		)

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		if err != nil {
			logger.Error("grpc call failed",
				zap.String("method", method),
				zap.String("target", cc.Target()),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else {
			logger.Debug("grpc call completed",
				zap.String("method", method),
				zap.String("target", cc.Target()),
				zap.Duration("duration", duration),
			)
		}

		return err
	}
}

// LoggingStreamInterceptor 创建流式 RPC 日志拦截器
func LoggingStreamInterceptor(logger *zap.Logger) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		start := time.Now()

		logger.Debug("grpc stream started",
			zap.String("method", method),
			zap.String("target", cc.Target()),
		)

		stream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			logger.Error("grpc stream failed to start",
				zap.String("method", method),
				zap.String("target", cc.Target()),
				zap.Duration("duration", time.Since(start)),
				zap.Error(err),
			)
			return nil, err
		}

		return &loggingStream{
			ClientStream: stream,
			logger:       logger,
			method:       method,
			target:       cc.Target(),
			startTime:    start,
		}, nil
	}
}

type loggingStream struct {
	grpc.ClientStream
	logger    *zap.Logger
	method    string
	target    string
	startTime time.Time
}

func (s *loggingStream) RecvMsg(m interface{}) error {
	err := s.ClientStream.RecvMsg(m)
	if err != nil {
		s.logger.Debug("grpc stream recv completed",
			zap.String("method", s.method),
			zap.String("target", s.target),
			zap.Duration("total_duration", time.Since(s.startTime)),
		)
	}
	return err
}

// MetadataUnaryInterceptor 创建一元 RPC 元数据拦截器
func MetadataUnaryInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		md := metadata.Pairs(
			"x-client-name", serviceName,
			"x-request-time", time.Now().Format(time.RFC3339Nano),
		)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// MetadataStreamInterceptor 创建流式 RPC 元数据拦截器
func MetadataStreamInterceptor(serviceName string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		md := metadata.Pairs(
			"x-client-name", serviceName,
			"x-request-time", time.Now().Format(time.RFC3339Nano),
		)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// RetryUnaryInterceptor 创建带重试的一元 RPC 拦截器
func RetryUnaryInterceptor(maxRetries int, retryDelay time.Duration, logger *zap.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error
		for i := 0; i <= maxRetries; i++ {
			if i > 0 {
				logger.Warn("retrying grpc call",
					zap.String("method", method),
					zap.Int("attempt", i+1),
					zap.Int("max_retries", maxRetries+1),
				)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(retryDelay):
				}
			}

			lastErr = invoker(ctx, method, req, reply, cc, opts...)
			if lastErr == nil {
				return nil
			}

			grpcErr, ok := ConvertError(lastErr)
			if !ok || !grpcErr.Retryable {
				return lastErr
			}
		}
		return lastErr
	}
}
