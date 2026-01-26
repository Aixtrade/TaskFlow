package tasktype

type Type string

const (
	// Demo 演示任务
	Demo Type = "demo"

	// GRPCTask 通用 gRPC 流式任务
	// 可调用任何实现了 TaskExecutorService 接口的服务
	GRPCTask Type = "grpc_task"
)

func (t Type) String() string {
	return string(t)
}

func (t Type) Queue() string {
	return "default"
}

func (t Type) IsValid() bool {
	switch t {
	case Demo, GRPCTask:
		return true
	default:
		return false
	}
}

var AllTypes = []Type{
	Demo,
	GRPCTask,
}
