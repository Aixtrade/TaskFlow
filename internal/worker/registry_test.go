package worker

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/pkg/tasktype"
)

type dummyHandler struct {
	name string
}

func (d dummyHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	return nil
}

func (d dummyHandler) Type() string {
	return d.name
}

func TestRegistryRegisterAndHasHandler(t *testing.T) {
	registry := NewRegistry(zap.NewNop())
	registry.Register(dummyHandler{name: tasktype.Demo.String()})

	if !registry.HasHandler(tasktype.Demo) {
		t.Fatal("expected handler for demo task")
	}
}

func TestRegistryTypes(t *testing.T) {
	registry := NewRegistry(zap.NewNop())
	registry.Register(dummyHandler{name: "a"})
	registry.Register(dummyHandler{name: "b"})

	types := registry.Types()
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
}
