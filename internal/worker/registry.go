package worker

import (
	"go.uber.org/zap"

	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type Registry struct {
	handlers map[string]Handler
	logger   *zap.Logger
}

func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		handlers: make(map[string]Handler),
		logger:   logger,
	}
}

func (r *Registry) Register(handler Handler) {
	r.handlers[handler.Type()] = handler
	r.logger.Info("registered handler", zap.String("type", handler.Type()))
}

func (r *Registry) RegisterAll(handlers ...Handler) {
	for _, h := range handlers {
		r.Register(h)
	}
}

func (r *Registry) Get(taskType string) (Handler, bool) {
	h, ok := r.handlers[taskType]
	return h, ok
}

func (r *Registry) Types() []string {
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

func (r *Registry) SetupServer(server *asynqqueue.Server) {
	for taskType, handler := range r.handlers {
		server.HandleFunc(taskType, handler.ProcessTask)
	}
}

func (r *Registry) HasHandler(taskType tasktype.Type) bool {
	_, ok := r.handlers[taskType.String()]
	return ok
}
