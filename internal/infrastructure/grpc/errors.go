package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCError 表示 gRPC 调用错误
type GRPCError struct {
	Code      string
	Message   string
	Retryable bool
}

// Error 实现 error 接口
func (e *GRPCError) Error() string {
	return e.Message
}

// ConvertError 将 gRPC 错误转换为 GRPCError
// 返回转换后的错误和是否成功转换的标志
func ConvertError(err error) (*GRPCError, bool) {
	if err == nil {
		return nil, false
	}

	st, ok := status.FromError(err)
	if !ok {
		return &GRPCError{
			Code:      "UNKNOWN",
			Message:   err.Error(),
			Retryable: true, // 未知错误默认可重试
		}, true
	}

	grpcErr := &GRPCError{
		Code:      st.Code().String(),
		Message:   st.Message(),
		Retryable: isRetryable(st.Code()),
	}

	return grpcErr, true
}

// isRetryable 根据 gRPC 状态码判断是否可重试
func isRetryable(code codes.Code) bool {
	switch code {
	case codes.Unavailable,
		codes.ResourceExhausted,
		codes.Aborted,
		codes.DeadlineExceeded,
		codes.Internal:
		return true
	case codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.FailedPrecondition,
		codes.Unimplemented,
		codes.Unauthenticated:
		return false
	default:
		return true
	}
}

// IsConnectionError 检查是否为连接错误
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return st.Code() == codes.Unavailable
}

// IsTimeoutError 检查是否为超时错误
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return st.Code() == codes.DeadlineExceeded
}
